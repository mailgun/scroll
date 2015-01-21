package registry

import (
	"github.com/mailgun/log"
	"github.com/mailgun/scroll/vulcan/middleware"
)

// SkipRegistry is an implementation of Registry for applications that do not need service discovery.
type SkipRegistry struct {
}

// RegisterApp is a no-op.
func (s *SkipRegistry) RegisterApp(name string, host string, port int) error {
	log.Infof("Skipping application registration for SkipRegistry")
	return nil
}

// RegisterHandler is a no-op.
func (s *SkipRegistry) RegisterHandler(name string, host string, path string, methods []string, middlewares []middleware.Middleware) error {
	log.Infof("Skipping handler registration for SkipRegistry")
	return nil
}
