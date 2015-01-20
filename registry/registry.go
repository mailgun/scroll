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

// RegistrationStrategy is an interface that all built-in and user-defined registration strategies implement.
type RegistrationStrategy interface {
	RegisterApp(registration *AppRegistration) error
	RegisterHandler(registration *HandlerRegistration) error
}

// Heartbeater periodically registers an application using the provided RegistrationStrategy.
type Heartbeater struct {
	ticker       *time.Ticker
	registration *AppRegistration
	strategy     RegistrationStrategy
}

// NewHeartbeater creates a Heartbeater from the provided app and strategy.
func NewHeartbeater(registration *AppRegistration, strategy RegistrationStrategy) *Heartbeater {
	return &Heartbeater{registration: registration, strategy: strategy}
}

// Start begins sending heartbeats.
func (h *Heartbeater) Start() {
	h.ticker = time.NewTicker(10 * time.Millisecond)
	go h.heartbeat()
}

// Stop halts sending heartbeats.
func (h *Heartbeater) Stop() {
	h.ticker.Stop()
}

func (h *Heartbeater) heartbeat() {
	for _ = range h.ticker.C {
		h.strategy.RegisterApp(h.registration)
	}
}