package registry

import (
	"fmt"

	"github.com/mailgun/go-etcd/etcd"
)

const (
	endpointKey = "vulcand/upstreams/%v/endpoints/%v"
	locationKey = "vulcand/hosts/%v/locations/%v"

	// If vulcand registration is enabled, the app will be re-registering itself every
	// this amount of seconds.
	endpointTTL = 60 // seconds
)

type Registry struct {
	etcdClient *etcd.Client
	config     Config
}

type Config struct {
	PublicAPIHost    string
	ProtectedAPIHost string
}

func NewRegistry(config Config) *Registry {
	return &Registry{
		etcdClient: etcd.NewClient([]string{"http://127.0.0.1:4001"}),
		config:     config,
	}
}

func (r *Registry) RegisterEndpoint(e *Endpoint) error {
	key := fmt.Sprintf(endpointKey, e.Name, e.ID)

	if _, err := r.etcdClient.Set(key, e.URL, endpointTTL); err != nil {
		return err
	}

	return nil
}

// RegisterLocation registers a provided location in vulcand.
func (r *Registry) RegisterLocation(l *Location) error {
	for _, scope := range l.Scope {
		apiHost, err := r.apiHostForScope(scope)
		if err != nil {
			return err
		}
		if err = r.registerLocation(l, apiHost); err != nil {
			return err
		}
	}
	return nil
}

// registerLocation is a helper that registers a provided location for a specified API hostname.
func (r *Registry) registerLocation(l *Location, apiHost string) error {
	key := fmt.Sprintf(locationKey, apiHost, l.ID)

	pathKey := fmt.Sprintf("%v/path", key)
	if _, err := r.etcdClient.Set(pathKey, l.Path, 0); err != nil {
		return err
	}

	upstreamKey := fmt.Sprintf("%v/upstream", key)
	if _, err := r.etcdClient.Set(upstreamKey, l.Upstream, 0); err != nil {
		return err
	}

	return nil
}

// apiHostForScope if a helper that returns an appropriate API hostname for a provided scope.
func (r *Registry) apiHostForScope(scope Scope) (string, error) {
	if scope == ScopePublic {
		return r.config.PublicAPIHost, nil
	} else if scope == ScopeProtected {
		return r.config.ProtectedAPIHost, nil
	} else {
		return "", fmt.Errorf("unknown scope value: %v", scope)
	}
}
