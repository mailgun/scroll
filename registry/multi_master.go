package registry

import "github.com/mailgun/scroll/vulcan"

/*
MultiMasterRegistry is an implementation of Registry in which multiple instances
of an application are able to accept requests at the same time. Internally, this
registry uses SingleMasterRegistry by varying the ID of server is registers.
*/
type MultiMasterRegistry struct {
	innerRegistry *SingleMasterRegistry
}

// NewMultiMasterRegistry creates a new MultiMasterRegistry from the provided etcd Client.
func NewMultiMasterRegistry(key string, ttl uint64) *MultiMasterRegistry {
	singleMasterRegistry := NewSingleMasterRegistry(key, ttl)

	return &MultiMasterRegistry{innerRegistry: singleMasterRegistry}
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *MultiMasterRegistry) RegisterApp(registration *AppRegistration) error {
	endpoint, err := vulcan.NewEndpoint(registration.Name, registration.Host, registration.Port)
	if err != nil {
		return nil
	}

	err = s.innerRegistry.registerBackend(endpoint)
	if err != nil {
		return err
	}

	err = s.innerRegistry.registerServer(endpoint)
	if err != nil {
		return err
	}

	return nil
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *MultiMasterRegistry) RegisterHandler(registration *HandlerRegistration) error {
	return s.innerRegistry.RegisterHandler(registration)
}
