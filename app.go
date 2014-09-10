package scroll

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/mailgun/gotools-log"
	"github.com/mailgun/manners"
	"github.com/mailgun/metrics"
)

const (
	DefaultLimit int = 100
	MaxLimit     int = 10000
	MaxBatchSize int = 1000
)

type App struct {
	config   *AppConfig
	router   *mux.Router
	registry *Registry

	Stats *AppStats
}

type AppConfig struct {
	Name     string
	Host     string
	Port     int
	APIHost  string
	Register bool
	Metrics  metrics.Metrics
}

func NewAppWithConfig(config *AppConfig) *App {
	var registry *Registry
	if config.Register != false {
		registry = NewRegistry()
	}

	return &App{
		config:   config,
		router:   mux.NewRouter(),
		registry: registry,
		Stats:    NewAppStats(config.Metrics),
	}
}

func (app *App) AddHandler(fn HandlerFunc, config *HandlerConfig) {
	handler := MakeHandler(app, fn, config)

	route := app.router.HandleFunc(config.Path, handler).Methods(config.Methods...)
	if len(config.Headers) != 0 {
		route.Headers(config.Headers...)
	}

	if app.registry != nil && config.Register != false {
		app.registerLocation(config.Methods, config.Path)
	}
}

func (app *App) AddHandlerWithBody(fn HandlerWithBodyFunc, config *HandlerConfig) {
	handler := MakeHandlerWithBody(app, fn, config)

	route := app.router.HandleFunc(config.Path, handler).Methods(config.Methods...)
	if len(config.Headers) != 0 {
		route.Headers(config.Headers...)
	}

	if app.registry != nil && config.Register != false {
		app.registerLocation(config.Methods, config.Path)
	}
}

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

	return manners.ListenAndServe(fmt.Sprintf("%v:%v", app.config.Host, app.config.Port), nil)
}

func (app *App) registerEndpoint() error {
	endpoint := NewEndpoint(app.config.Name, app.config.Host, app.config.Port)

	if err := app.registry.RegisterEndpoint(endpoint); err != nil {
		return err
	}

	log.Infof("Registered endpoint: %v", endpoint)

	return nil
}

func (app *App) registerLocation(methods []string, path string) error {
	location := NewLocation(app.config.APIHost, methods, path, app.config.Name)

	if err := app.registry.RegisterLocation(location); err != nil {
		return err
	}

	log.Infof("Registered location: %v", location)

	return nil
}
