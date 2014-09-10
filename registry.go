package scroll

import (
	"fmt"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

const (
	EndpointKey = "vulcand/upstreams/%v/endpoints/%v"
	LocationKey = "vulcand/hosts/%v/locations/%v"

	EndpointTTL = 60
)

type Registry struct {
	etcdClient *etcd.Client
}

type Endpoint struct {
	upstream string
	host     string
	port     int
}

type Location struct {
	apiHost  string
	methods  []string
	path     string
	upstream string
}

func NewRegistry() *Registry {
	return &Registry{
		etcdClient: etcd.NewClient([]string{"http://127.0.0.1:4001"}),
	}
}

func (r *Registry) RegisterEndpoint(e *Endpoint) error {
	key := fmt.Sprintf(EndpointKey, e.GetUpstream(), e.GetID())

	if _, err := r.etcdClient.Set(key, e.GetEndpoint(), EndpointTTL); err != nil {
		return err
	}

	return nil
}

func (r *Registry) RegisterLocation(l *Location) error {
	key := fmt.Sprintf(LocationKey, l.GetAPIHost(), l.GetID())

	pathKey := fmt.Sprintf("%v/path", key)
	if _, err := r.etcdClient.Set(pathKey, l.GetPath(), 0); err != nil {
		return err
	}

	upstreamKey := fmt.Sprintf("%v/upstream", key)
	if _, err := r.etcdClient.Set(upstreamKey, l.GetUpstream(), 0); err != nil {
		return err
	}

	return nil
}

func NewEndpoint(upstream, host string, port int) *Endpoint {
	return &Endpoint{
		upstream: upstream,
		host:     host,
		port:     port,
	}
}

func (e *Endpoint) GetID() string {
	return fmt.Sprintf("%v_%v", e.host, e.port)
}

func (e *Endpoint) GetUpstream() string {
	return e.upstream
}

func (e *Endpoint) GetEndpoint() string {
	return fmt.Sprintf("http://%v:%v", e.host, e.port)
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("id [%v], upstream [%v], endpoint [%v]",
		e.GetID(), e.GetUpstream(), e.GetEndpoint())
}

func NewLocation(apiHost string, methods []string, path, upstream string) *Location {
	return &Location{
		apiHost:  apiHost,
		methods:  methods,
		path:     convertPath(path),
		upstream: upstream,
	}
}

func (l *Location) GetID() string {
	return strings.Replace(fmt.Sprintf("%v%v", strings.Join(l.methods, "."), l.path), "/", ".", -1)
}

func (l *Location) GetAPIHost() string {
	return l.apiHost
}

func (l *Location) GetPath() string {
	methods := strings.Join(l.methods, `", "`)
	return fmt.Sprintf(`TrieRoute("%v", "%v")`, methods, l.path)
}

func (l *Location) GetUpstream() string {
	return l.upstream
}

func (l *Location) String() string {
	return fmt.Sprintf("id [%v], API host [%v], path [%v], upstream [%v]",
		l.GetID(), l.GetAPIHost(), l.GetPath(), l.GetUpstream())
}

func convertPath(path string) string {
	return strings.Replace(strings.Replace(path, "{", "<", -1), "}", ">", -1)
}
