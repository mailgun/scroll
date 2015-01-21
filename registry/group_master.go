package registry

import (
	"encoding/json"
	"fmt"

	"github.com/mailgun/go-etcd/etcd"
	"github.com/mailgun/scroll/vulcan"
)

const (
	frontendKey   = "%s/frontends/%s.%s/frontend"
	middlewareKey = "%s/frontends/%s.%s/middlewares/%s"
	backendKey    = "%s/backends/%s/backend"
	serverKey     = "%s/backends/%s/servers/%s"
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
	Client   *etcd.Client
}

// NewGroupMasterRegistry creates a new GroupMasterRegistry from the provided etcd Client.
func NewGroupMasterRegistry(key string, group string, ttl uint64) *GroupMasterRegistry {
	client := etcd.NewClient([]string{"http://127.0.0.1:4001"})

	return &GroupMasterRegistry{
		Key:      key,
		Group:    group,
		TTL:      ttl,
		Client:   client,
		IsMaster: false,
	}
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *GroupMasterRegistry) RegisterApp(registration *AppRegistration) error {
	endpoint, err := vulcan.NewEndpointWithID(s.Group, registration.Name, registration.Host, registration.Port)
	if err != nil {
		return nil
	}

	err = s.registerBackend(endpoint)
	if err != nil {
		return err
	}

	err = s.registerServer(endpoint)
	if err != nil {
		return err
	}

	return nil
}

func (s *GroupMasterRegistry) registerBackend(endpoint *vulcan.Endpoint) error {
	key := fmt.Sprintf(backendKey, s.Key, endpoint.Name)
	backend, err := endpoint.BackendSpec()
	if err != nil {
		return err
	}

	_, err = s.Client.Set(key, backend, 0)
	if err != nil {
		return err
	}

	return err
}

func (s *GroupMasterRegistry) registerServer(endpoint *vulcan.Endpoint) error {
	if s.IsMaster {
		return s.maintainMasterRole(endpoint)
	}

	return s.assumeMasterRole(endpoint)
}

func (s *GroupMasterRegistry) assumeMasterRole(endpoint *vulcan.Endpoint) error {
	key := fmt.Sprintf(serverKey, s.Key, endpoint.Name, endpoint.ID)
	server, err := endpoint.ServerSpec()
	if err != nil {
		return nil
	}

	_, err = s.Client.Create(key, server, s.TTL)
	if err != nil {
		return err
	}

	s.IsMaster = true

	return nil
}

func (s *GroupMasterRegistry) maintainMasterRole(endpoint *vulcan.Endpoint) error {
	key := fmt.Sprintf(serverKey, s.Key, endpoint.Name, endpoint.ID)
	server, err := endpoint.ServerSpec()
	if err != nil {
		return nil
	}

	_, err = s.Client.CompareAndSwap(key, server, s.TTL, server, 0)
	if err != nil {
		s.IsMaster = false
		return err
	}

	s.IsMaster = true
	return nil
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *GroupMasterRegistry) RegisterHandler(registration *HandlerRegistration) error {
	location := vulcan.NewLocation(registration.Host, registration.Methods, registration.Path, registration.Name, registration.Middlewares)
	err := s.registerFrontend(location)
	if err != nil {
		return err
	}

	err = s.registerMiddleware(location)
	if err != nil {
		return err
	}

	return nil
}

func (s *GroupMasterRegistry) registerFrontend(location *vulcan.Location) error {
	key := fmt.Sprintf(frontendKey, s.Key, location.Host, location.ID)
	frontend, err := location.Spec()
	if err != nil {
		return err
	}

	_, err = s.Client.Set(key, frontend, 0)
	if err != nil {
		return err
	}

	return nil
}

func (s *GroupMasterRegistry) registerMiddleware(location *vulcan.Location) error {
	for i, m := range location.Middlewares {
		m.Priority = i

		key := fmt.Sprintf(middlewareKey, s.Key, location.Host, location.ID, m.ID)
		middleware, err := json.Marshal(m)
		if err != nil {
			return err
		}

		_, err = s.Client.Set(key, string(middleware), 0)
		if err != nil {
			return err
		}
	}

	return nil
}
