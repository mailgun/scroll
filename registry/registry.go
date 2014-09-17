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
}

func NewRegistry() *Registry {
	return &Registry{
		etcdClient: etcd.NewClient([]string{"http://127.0.0.1:4001"}),
	}
}

func (r *Registry) RegisterEndpoint(e *Endpoint) error {
	key := fmt.Sprintf(endpointKey, e.Name, e.ID)

	if _, err := r.etcdClient.Set(key, e.URL, endpointTTL); err != nil {
		return err
	}

	return nil
}

func (r *Registry) RegisterLocation(l *Location) error {
	key := fmt.Sprintf(locationKey, l.APIHost, l.ID)

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
