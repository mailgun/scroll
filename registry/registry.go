package registry

import "github.com/mailgun/scroll/vulcan/middleware"

// RegistrationStrategy is an interface that all built-in and user-defined registration strategies implement.
type RegistrationStrategy interface {
	RegisterApp(name string, host string, port int) error
	RegisterHandler(name string, host string, path string, methods []string, middlewares []middleware.Middleware) error
}
