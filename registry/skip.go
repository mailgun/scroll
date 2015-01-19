package registry

import (
	"github.com/mailgun/log"
	"github.com/mailgun/scroll/vulcan/middleware"
)

// SkipStrategy is an implementation of Strategy for applications that do not need service discovery.
type SkipStrategy struct {
}

// RegisterApp is a no-op.
func (s *SkipStrategy) RegisterApp(name string, host string, port int) error {
	log.Infof("Skipping application registration for NoRegistrationStrategy registry")
	return nil
}

// RegisterHandler is a no-op.
func (s *SkipStrategy) RegisterHandler(name string, host string, path string, methods []string, middlewares []middleware.Middleware) error {
	log.Infof("Skipping handler registration for NoRegistrationStrategy registry")
	return nil
}
