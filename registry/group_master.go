package registry

import (
	"github.com/mailgun/log"
	"github.com/mailgun/scroll/vulcan"
)

/*
GroupMasterRegistry is an implementation of Registry that uses a single master
instance of an application within a given group to handle requests. When the
master instance fails, request handling will automatically failover to a slave
instance.
*/
type GroupMasterRegistry struct {
	Key      string
	Group    string
	TTL      uint64
	IsMaster bool
	client   *vulcan.Client
}

// NewGroupMasterRegistry creates a new GroupMasterRegistry from the provided etcd Client.
func NewGroupMasterRegistry(key string, group string, ttl uint64) *GroupMasterRegistry {
	client := vulcan.NewClient(key)

	return &GroupMasterRegistry{
		Key:      key,
		Group:    group,
		TTL:      ttl,
		client:   client,
		IsMaster: false,
	}
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *GroupMasterRegistry) RegisterApp(registration *AppRegistration) error {
	log.Infof("Registering app: %v", registration)

	endpoint, err := vulcan.NewEndpointWithID(s.Group, registration.Name, registration.Host, registration.Port)
	if err != nil {
		return err
	}

	err = s.client.RegisterBackend(endpoint)
	if err != nil {
		log.Errorf("Failed to register backend for endpoint: %v, %s", endpoint, err)
		return err
	}

	if s.IsMaster {
		err = s.maintainLeader(endpoint)
	} else {
		err = s.initLeader(endpoint)
	}

	if err != nil {
		log.Errorf("Failed to register server for endpoint: %v, %s", endpoint, err)
		return err
	}

	return nil
}

func (s *GroupMasterRegistry) initLeader(endpoint *vulcan.Endpoint) error {
	err := s.client.CreateServer(endpoint, s.TTL)
	if err != nil {
		return err
	}

	log.Infof("Assumed master role for endpoint: %v", endpoint)
	s.IsMaster = true

	return nil
}

func (s *GroupMasterRegistry) maintainLeader(endpoint *vulcan.Endpoint) error {
	err := s.client.UpdateServer(endpoint, s.TTL)
	if err != nil {
		log.Infof("Falling back to follow role for endpoint: %v", endpoint)
		s.IsMaster = false
		return err
	}

	return nil
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *GroupMasterRegistry) RegisterHandler(registration *HandlerRegistration) error {
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
