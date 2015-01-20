package registry

import (
	"github.com/mailgun/go-etcd/etcd"
	"github.com/mailgun/scroll/vulcan"
	"github.com/mailgun/scroll/vulcan/middleware"
)

/*
MultiMasterStrategy is an implementation of RegistrationStrategy in which
multiple instances of an application are able to accept requests at the same
time. Internally, this strategy uses SingleMasterStrategy by varying the ID of
server is registers.
*/
type MultiMasterStrategy struct {
	innerStrategy *SingleMasterStrategy
}

// NewMultiMasterStrategy creates a new MultiMasterStrategy from the provided etcd Client.
func NewMultiMasterStrategy(key string, ttl uint64, client *etcd.Client) *MultiMasterStrategy {
	singleMasterStrategy := NewSingleMasterStrategy(key, ttl, client)

	return &MultiMasterStrategy{innerStrategy: singleMasterStrategy}
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *MultiMasterStrategy) RegisterApp(name string, host string, port int) error {
	endpoint, err := vulcan.NewEndpoint(name, host, port)
	if err != nil {
		return nil
	}

	err = s.innerStrategy.registerBackend(endpoint)
	if err != nil {
		return err
	}

	err = s.innerStrategy.registerServer(endpoint)
	if err != nil {
		return err
	}

	return nil
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *MultiMasterStrategy) RegisterHandler(name string, host string, path string, methods []string, middlewares []middleware.Middleware) error {
	return s.innerStrategy.RegisterHandler(name, host, path, methods, middlewares)
}
