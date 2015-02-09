package registry

import (
	"github.com/mailgun/log"
	"github.com/mailgun/scroll/vulcan"
)

/*
MultiMasterRegistry is an implementation of Registry in which multiple instances
of an application are able to accept requests at the same time. Internally, this
registry uses GroupMasterRegistry by creating a unique group ID for each
instance of an application.
*/
type MultiMasterRegistry struct {
	Key    string
	TTL    uint64
	client *vulcan.Client
}

// NewMultiMasterRegistry creates a new MultiMasterRegistry from the provided etcd Client.
func NewMultiMasterRegistry(key string, ttl uint64) (*MultiMasterRegistry, error) {
	client := vulcan.NewClient(key)

	return &MultiMasterRegistry{
		Key:    key,
		TTL:    ttl,
		client: client,
	}, nil
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *MultiMasterRegistry) RegisterApp(registration *AppRegistration) error {
	log.Infof("Registering app: %v", registration)

	endpoint, err := vulcan.NewEndpoint(registration.Name, registration.Host, registration.Port)
	if err != nil {
		return err
	}

	err = s.client.RegisterBackend(endpoint)
	if err != nil {
		log.Errorf("Failed to register backend for endpoint: %v, %s", endpoint, err)
		return err
	}

	err = s.client.UpsertServer(endpoint, s.TTL)
	if err != nil {
		log.Errorf("Failed to register server for endpoint: %v, %s", endpoint, err)
		return err
	}

	return nil
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *MultiMasterRegistry) RegisterHandler(registration *HandlerRegistration) error {
	log.Infof("Registering handler: %v", registration)

	location := vulcan.NewLocation(registration.Host, registration.Methods, registration.Path, registration.Name, registration.Middlewares)
	err := s.client.RegisterFrontend(location)
	if err != nil {
		log.Errorf("Failed to register frontend for location: %v, %s", location, err)
		return err
	}

	err = s.client.RegisterMiddleware(location)
	if err != nil {
		log.Errorf("Failed to register middleware for location: %v, %s", location, err)
		return err
	}

	return nil
}
