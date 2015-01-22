package registry

import (
	"time"

	"github.com/mailgun/scroll/vulcan/middleware"
)

// AppRegistration contains data about an app to be registered.
type AppRegistration struct {
	Name string
	Host string
	Port int
}

// HandlerRegistration contains data about a handler to be registered.
type HandlerRegistration struct {
	Name        string
	Host        string
	Path        string
	Methods     []string
	Middlewares []middleware.Middleware
}

// Registry is an interface that all built-in and user-defined registries implement.
type Registry interface {
	RegisterApp(registration *AppRegistration) error
	RegisterHandler(registration *HandlerRegistration) error
}

// Heartbeater periodically registers an application using the provided Registry.
type Heartbeater struct {
	Running      bool
	ticker       *time.Ticker
	registration *AppRegistration
	registry     Registry
	interval     time.Duration
}

// NewHeartbeater creates a Heartbeater from the provided app and registry.
func NewHeartbeater(registration *AppRegistration, registry Registry, interval time.Duration) *Heartbeater {
	return &Heartbeater{registration: registration, registry: registry, interval: interval}
}

// Start begins sending heartbeats.
func (h *Heartbeater) Start() {
	h.Running = true
	h.ticker = time.NewTicker(h.interval)
	go h.heartbeat()
}

// Stop halts sending heartbeats.
func (h *Heartbeater) Stop() {
	h.ticker.Stop()
	h.Running = false
}

// Toggle starts or stops the Heartbeater based on whether it is already running.
func (h *Heartbeater) Toggle() {
	if h.Running {
		h.Stop()
	} else {
		h.Start()
	}
}

func (h *Heartbeater) heartbeat() {
	for range h.ticker.C {
		h.registry.RegisterApp(h.registration)
	}
}
