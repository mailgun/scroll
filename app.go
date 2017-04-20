package scroll

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mailgun/log"
	"github.com/mailgun/metrics"
	"github.com/mailgun/scroll/vulcand"
)

const (
	// Suggested result set limit for APIs that may return many entries (e.g. paging).
	DefaultLimit = 100

	// Suggested max allowed result set limit for APIs that may return many entries (e.g. paging).
	MaxLimit = 10000

	// Suggested max allowed amount of entries that batch APIs can accept (e.g. batch uploads).
	MaxBatchSize = 1000

	defaultHTTPReadTimeout  = 10 * time.Second
	defaultHTTPWriteTimeout = 60 * time.Second
	defaultHTTPIdleTimeout  = 60 * time.Second
)

// Represents an app.
type App struct {
	Config     AppConfig
	router     *mux.Router
	stats      *appStats
	vulcandReg *vulcand.Registry
	wg         sync.WaitGroup
	exitCh     chan os.Signal
}

// Represents a configuration object an app is created with.
type AppConfig struct {
	// name of the app being created
	Name string

	// IP/port the app will bind to
	ListenIP   string
	ListenPort int

	// optional router to use
	Router *mux.Router

	// hostnames of the public and protected API entrypoints used for vulcan registration
	PublicAPIHost    string
	ProtectedAPIHost string
	ProtectedAPIURL  string

	// Vulcand config must be provided to enable registration in Etcd.
	Vulcand *vulcand.Config

	// metrics service used for emitting the app's real-time metrics
	Client metrics.Client

	HTTP struct {
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}
}

// Create a new app.
func NewApp() (*App, error) {
	return NewAppWithConfig(AppConfig{})
}

// Create a new app with the provided configuration.
func NewAppWithConfig(config AppConfig) (*App, error) {
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = defaultHTTPReadTimeout
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = defaultHTTPWriteTimeout
	}
	if config.HTTP.IdleTimeout == 0 {
		config.HTTP.IdleTimeout = defaultHTTPIdleTimeout
	}
	app := App{Config: config}

	if LogRequest == nil {
		LogRequest = logRequest
	}

	app.router = config.Router
	if app.router == nil {
		app.router = mux.NewRouter()
		app.router.UseEncodedPath()
	}
	app.router.HandleFunc("/_ping", handlePing).Methods("GET")

	if config.Vulcand != nil {
		var err error
		app.vulcandReg, err = vulcand.NewRegistry(*config.Vulcand, config.Name, config.ListenIP, config.ListenPort)
		if err != nil {
			return nil, err
		}
	}

	app.stats = newAppStats(config.Client)
	return &app, nil
}

// Register a handler function.
//
// If vulcan registration is enabled in the both app config and handler spec,
// the handler will be registered in the local etcd instance.
func (app *App) AddHandler(spec Spec) error {
	var handler http.HandlerFunc

	// make a handler depending on the function provided in the spec
	if spec.RawHandler != nil {
		handler = spec.RawHandler
	} else if spec.Handler != nil {
		handler = MakeHandler(app, spec.Handler, spec)
	} else if spec.HandlerWithBody != nil {
		handler = MakeHandlerWithBody(app, spec.HandlerWithBody, spec)
	} else {
		return fmt.Errorf("the spec does not provide a handler function: %v", spec)
	}

	for _, path := range spec.Paths {
		route := app.router.HandleFunc(path, handler).Methods(spec.Methods...)
		if len(spec.Headers) != 0 {
			route.Headers(spec.Headers...)
		}
		if app.vulcandReg != nil {
			app.registerFrontend(spec.Methods, path, spec.Scope, spec.Middlewares)
		}
	}

	return nil
}

// GetHandler returns HTTP compatible Handler interface.
func (app *App) GetHandler() http.Handler {
	return app.router
}

// SetNotFoundHandler sets the handler for the case when URL can not be matched by the router.
func (app *App) SetNotFoundHandler(fn http.HandlerFunc) {
	app.router.NotFoundHandler = fn
}

// IsPublicRequest determines whether the provided request came through the public HTTP endpoint.
func (app *App) IsPublicRequest(request *http.Request) bool {
	return request.Host == app.Config.PublicAPIHost
}

// Start the app on the configured host/port.
//
// Supports graceful shutdown on 'kill' and 'int' signals.
func (app *App) Run() error {
	if app.vulcandReg != nil {
		err := app.vulcandReg.Start()
		if err != nil {
			return fmt.Errorf("failed to start vulcand registry: err=(%s)", err)
		}
		heartbeatCh := make(chan os.Signal, 1)
		signal.Notify(heartbeatCh, syscall.SIGUSR1)
		go func() {
			sig := <-heartbeatCh
			log.Infof("Got signal %v, canceling vulcand registration", sig)
			app.vulcandReg.Stop()
		}()
	}

	addr := fmt.Sprintf("%v:%v", app.Config.ListenIP, app.Config.ListenPort)
	httpSrv := &http.Server{
		Addr:         addr,
		ReadTimeout:  app.Config.HTTP.ReadTimeout,
		WriteTimeout: app.Config.HTTP.WriteTimeout,
		IdleTimeout:  app.Config.HTTP.IdleTimeout,
		Handler:      app.router,
	}

	// listen for a shutdown signal
	app.exitCh = make(chan os.Signal, 1)
	signal.Notify(app.exitCh, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		app.wg.Add(1)
		if s, ok := <-app.exitCh; ok {
			log.Infof("Got signal %v, shutting down", s)
		}
		if app.vulcandReg != nil {
			app.vulcandReg.Stop()
		}
		httpSrv.Shutdown(context.Background())
		app.wg.Done()
	}()
	return httpSrv.ListenAndServe()
}

// Stops the http server and un-registers with vulcand
func (app *App) Stop() {
	if app.exitCh != nil {
		close(app.exitCh)
	}
	app.wg.Wait()
}

// registerLocation is a helper for registering handlers in vulcan.
func (app *App) registerFrontend(methods []string, path string, scope Scope, middlewares []vulcand.Middleware) error {
	host, err := app.apiHostForScope(scope)
	if err != nil {
		return err
	}
	app.vulcandReg.AddFrontend(host, path, methods, middlewares)
	return nil
}

// apiHostForScope is a helper that returns an appropriate API hostname for a provided scope.
func (app *App) apiHostForScope(scope Scope) (string, error) {
	if scope == ScopePublic {
		return app.Config.PublicAPIHost, nil
	} else if scope == ScopeProtected {
		return app.Config.ProtectedAPIHost, nil
	} else {
		return "", fmt.Errorf("unknown scope value: %v", scope)
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
