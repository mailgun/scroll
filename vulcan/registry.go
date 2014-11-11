package vulcan

import (
	"encoding/json"
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

// RegisterLocation registers a provided location in vulcan.
func (r *Registry) RegisterLocation(l *Location) error {
	key := fmt.Sprintf(locationKey, l.Host, l.ID)

	pathKey := fmt.Sprintf("%v/path", key)
	if _, err := r.etcdClient.Set(pathKey, l.Path, 0); err != nil {
		return err
	}

	upstreamKey := fmt.Sprintf("%v/upstream", key)
	if _, err := r.etcdClient.Set(upstreamKey, l.Upstream, 0); err != nil {
		return err
	}

	options, _ := json.Marshal(l.Options)
	optionsKey := fmt.Sprintf("%v/options", key)
	if _, err := r.etcdClient.Set(optionsKey, string(options), 0); err != nil {
		return err
	}

	for _, m := range l.Middlewares {
		middleware, _ := json.Marshal(m)
		middlewareKey := fmt.Sprintf("%v/middlewares/%v/%v", key, m.Type, m.ID)
		if _, err := r.etcdClient.Set(middlewareKey, string(middleware), 0); err != nil {
			return err
		}
	}

	return nil
}
