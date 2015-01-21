package registry

import (
	"fmt"
	"os"
)

/*
MultiMasterRegistry is an implementation of Registry in which multiple instances
of an application are able to accept requests at the same time. Internally, this
registry uses GroupMasterRegistry by creating a unique group ID for each
instance of an application.
*/
type MultiMasterRegistry struct {
	innerRegistry *GroupMasterRegistry
}

// NewMultiMasterRegistry creates a new MultiMasterRegistry from the provided etcd Client.
func NewMultiMasterRegistry(key string, port int, ttl uint64) (*MultiMasterRegistry, error) {
	group, err := makeGroupID(port)
	if err != nil {
		return nil, err
	}

	innerRegistry := NewGroupMasterRegistry(key, group, ttl)

	return &MultiMasterRegistry{innerRegistry: innerRegistry}, nil
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *MultiMasterRegistry) RegisterApp(registration *AppRegistration) error {
	return s.innerRegistry.RegisterApp(registration)
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *MultiMasterRegistry) RegisterHandler(registration *HandlerRegistration) error {
	return s.innerRegistry.RegisterHandler(registration)
}

func makeGroupID(listenPort int) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v_%v", hostname, listenPort), nil
}
