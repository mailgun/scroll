package registry

import "github.com/mailgun/go-etcd/etcd"

const (
	masterNodeID = "master"
)

/*
SingleMasterRegistry is an implementation of Registry that uses a single master
instance of an application to handle requests. Internally, this registry uses
a GroupMasterRegistry with a single, constant group ID.
*/
type SingleMasterRegistry struct {
	Key           string
	TTL           uint64
	IsMaster      bool
	Client        *etcd.Client
	innerRegistry *GroupMasterRegistry
}

// NewSingleMasterRegistry creates a new SingleMasterRegistry from the provided etcd Client.
func NewSingleMasterRegistry(key string, ttl uint64) *SingleMasterRegistry {
	client := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	innerRegistry := NewGroupMasterRegistry(key, masterNodeID, ttl)

	return &SingleMasterRegistry{
		Key:           key,
		TTL:           ttl,
		Client:        client,
		IsMaster:      false,
		innerRegistry: innerRegistry,
	}
}

// RegisterApp adds a new backend and a single server with Vulcand.
func (s *SingleMasterRegistry) RegisterApp(r *AppRegistration) error {
	return s.innerRegistry.RegisterApp(r)
}

// RegisterHandler registers the frontends and middlewares with Vulcand.
func (s *SingleMasterRegistry) RegisterHandler(r *HandlerRegistration) error {
	return s.innerRegistry.RegisterHandler(r)
}
