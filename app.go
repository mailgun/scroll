package scroll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mailgun/log"
	"github.com/mailgun/manners"
	"github.com/mailgun/metrics"

	"github.com/mailgun/scroll/registry"
)

const (
	// Suggested result set limit for APIs that may return many entries (e.g. paging).
	DefaultLimit int = 100

	// Suggested max allowed result set limit for APIs that may return many entries (e.g. paging).
	MaxLimit int = 10000

	// Suggested max allowed amount of entries that batch APIs can accept (e.g. batch uploads).
	MaxBatchSize int = 1000
)

// Represents an app.
type App struct {
	config   AppConfig
	router   *mux.Router
	registry *registry.Registry
	stats    *appStats
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

	// hostnames of the public and protected API entrypoints used for vulcand registration
	PublicAPIHost    string
	ProtectedAPIHost string

	// whether to register the app's endpoint and handlers in vulcand
	Register bool

	// metrics service used for emitting the app's real-time metrics
	Client metrics.Client
}

// Create a new app.
func NewApp() *App {
	return NewAppWithConfig(AppConfig{})
}

// Create a new app with the provided configuration.
func NewAppWithConfig(config AppConfig) *App {
	var reg *registry.Registry
	if config.Register != false {
		reg = registry.NewRegistry(registry.Config{
			PublicAPIHost:    config.PublicAPIHost,
			ProtectedAPIHost: config.ProtectedAPIHost,
		})
	}

	router := config.Router
	if router == nil {
		router = mux.NewRouter()
	}
	router.HandleFunc("/build_info", buildInfo).Methods("GET")

	return &App{
		config:   config,
		router:   router,
		registry: reg,
		stats:    newAppStats(config.Client),
	}
}

// Register a handler function.
//
// If vulcand registration is enabled in the both app config and handler spec,
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

	// register the handler in the router
	route := app.router.HandleFunc(spec.Path, handler).Methods(spec.Methods...)
	if len(spec.Headers) != 0 {
		route.Headers(spec.Headers...)
	}

	// vulcand registration
	if app.registry != nil && spec.Register != false {
		app.registerLocation(spec.Methods, spec.Path, spec.Scopes)
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

// Start the app on the configured host/port.
//
// If vulcand registration is enabled in the app config, starts a goroutine that
// will be registering the app's endpoint once every minute in the local etcd
// instance.
//
// Supports graceful shutdown on 'kill' and 'int' signals.
func (app *App) Run() error {
	http.Handle("/", app.router)

	if app.registry != nil {
		go func() {
			for {
				app.registerEndpoint()
				time.Sleep(60 * time.Second)
			}
		}()
	}

	// listen for a shutdown signal
	go func() {
		exitChan := make(chan os.Signal, 1)
		signal.Notify(exitChan, os.Interrupt, os.Kill)
		s := <-exitChan
		log.Infof("Got shutdown signal: %v", s)
		manners.Close()
	}()

	return manners.ListenAndServe(
		fmt.Sprintf("%v:%v", app.config.ListenIP, app.config.ListenPort), nil)
}

// registerEndpoint is a helper for registering the app's endpoint in vulcand.
func (app *App) registerEndpoint() {
	endpoint, err := registry.NewEndpoint(app.config.Name, app.config.ListenIP, app.config.ListenPort)
	if err != nil {
		log.Errorf("Failed to create an endpoint: %v", err)
		return
	}

	if err := app.registry.RegisterEndpoint(endpoint); err != nil {
		log.Errorf("Failed to register an endpoint: %v %v", endpoint, err)
		return
	}

	log.Infof("Registered: %v", endpoint)
}

// registerLocation is a helper for registering handlers in vulcand.
func (app *App) registerLocation(methods []string, path string, scopes []Scope) {
	for _, scope := range scopes {
		app.registerLocationForScope(methods, path, scope)
	}
}

// registerLocationForScope registers a location with a specified scope.
func (app *App) registerLocationForScope(methods []string, path string, scope Scope) {
	host, err := app.apiHostForScope(scope)
	if err != nil {
		log.Errorf("Failed to register a location: %v", err)
		return
	}
	app.registerLocationForHost(methods, path, host)
}

// registerLocationForHost registers a location for a specified hostname.
func (app *App) registerLocationForHost(methods []string, path, host string) {
	location := registry.NewLocation(host, methods, path, app.config.Name)

	if err := app.registry.RegisterLocation(location); err != nil {
		log.Errorf("Failed to register a location: %v %v", location, err)
		return
	}

	log.Infof("Registered: %v", location)
}

// apiHostForScope is a helper that returns an appropriate API hostname for a provided scope.
func (app *App) apiHostForScope(scope Scope) (string, error) {
	if scope == ScopePublic {
		return app.config.PublicAPIHost, nil
	} else if scope == ScopeProtected {
		return app.config.ProtectedAPIHost, nil
	} else {
		return "", fmt.Errorf("unknown scope value: %v", scope)
	}
}

// build is never explicitly initialized in code. It is only non-empty if the -X flag is set on the Go linker at build time.
var build string

// buildInfo responds to an http request with information about the current binary.
func buildInfo(w http.ResponseWriter, r *http.Request) {
	// If build information was set incorrectly or not at all during build time,
	// default to displaying the string "information missing" for all fields.
	empty := "information missing"
	info := struct {
		Commit      string `json:"commit"`
		Description string `json:"description"`
		GithubLink  string `json:"github link"`
		BuildTime   string `json:"build time"`
	}{empty, empty, empty, empty}

	// Marshall to json whatever we have in info when we exit the function.
	defer json.NewEncoder(w).Encode(&info)

	// Parse build. Expected format is:
	//    <commit hash> <commit message>; <date>; <location of package main>
	//
	// For example:
	//    e5469c7 tests passing when ldflags are provided; Fri Nov  7 16:05:28 PST 2014; github.com/mailgun/scroll
	parts := strings.Split(build, ";")
	if len(parts) != 3 {
		return
	}
	commit := strings.SplitN(strings.TrimSpace(parts[0]), " ", 2)
	if len(commit) != 2 {
		return
	}

	// Everything parsed successfully.
	info.Commit, info.Description = commit[0], commit[1]
	info.BuildTime = strings.TrimSpace(parts[1])
	info.GithubLink = "https://" + strings.TrimSpace(parts[2]) + "/commit/" + info.Commit

	return
}
