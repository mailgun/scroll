package vulcand

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/Shopify/toxiproxy/client"
	etcd "github.com/coreos/etcd/clientv3"
	. "gopkg.in/check.v1"
)

const testNamespace = "/test"

func TestRegistry(t *testing.T) {
	TestingT(t)
}

type RegistrySuite struct {
	client     *etcd.Client
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *Registry
	cfg        Config
	proxy      *toxiproxy.Proxy
}

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpSuite(c *C) {
	var err error

	// This assumes we used to `docker-compose up` to create the toxiproxy
	tox := toxiproxy.NewClient("localhost:8474")
	s.proxy, err = tox.CreateProxy("etcdv3", "proxy:22379", "etcd:2379")
	c.Assert(err, IsNil)

	// Send all etcd traffic through the proxy
	s.cfg = Config{
		Namespace: testNamespace,
		Etcd: &etcd.Config{
			Endpoints: []string{"https://localhost:22379"},
			Username:  "root",
			Password:  "rootpw",
			TLS: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		TTL: time.Second, // Set a short timeout so we can test keep alive disconnects
	}

	s.client, err = etcd.New(*s.cfg.Etcd)
	c.Assert(err, IsNil)
}

func (s *RegistrySuite) TearDownSuite(c *C) {
	s.proxy.Delete()
}

func (s *RegistrySuite) SetUpTest(c *C) {
	s.ctx, s.cancelFunc = context.WithTimeout(context.Background(), time.Second*20)
	_, err := s.client.Delete(s.ctx, testNamespace, etcd.WithPrefix())
	c.Assert(err, IsNil)
	s.r, err = NewRegistry(
		s.cfg,
		"app1",
		"192.168.19.2",
		8000)
	c.Assert(err, IsNil)
	err = s.r.Start()
	c.Assert(err, IsNil)

}

func (s *RegistrySuite) TearDownTest(c *C) {
	s.r.Stop()
	s.cancelFunc()
}

func (s *RegistrySuite) TestRegisterBackend(c *C) {
	bes, err := newBackendSpecWithID("foo", "bar", "example.com", 8000)
	c.Assert(err, IsNil)

	// When
	err = s.r.registerBackend(bes)

	// Then
	c.Assert(err, IsNil)

	res, err := s.client.Get(s.ctx, testNamespace+"/backends/bar/backend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/backends/bar/servers/foo")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"URL":"http://example.com:8000"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(s.r.leaseID))
}

func (s *RegistrySuite) TestRegisterFrontend(c *C) {
	m := []Middleware{{Type: "bar", ID: "bazz", Spec: "blah"}}
	fes := newFrontendSpec("foo", "host", "/path/to/server", []string{"GET"}, m)

	// When
	err := s.r.registerFrontend(fes)

	// Then
	c.Assert(err, IsNil)

	res, err := s.client.Get(s.ctx, testNamespace+"/frontends/host.get.path.to.server/frontend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http","BackendId":"foo","Route":"Host(\"host\") && Method(\"GET\") && Path(\"/path/to/server\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/host.get.path.to.server/middlewares/bazz")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"bar","Id":"bazz","Priority":0,"Middleware":"blah"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))
}

func (s *RegistrySuite) TestHeartbeat(c *C) {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode := res.Kvs[0]
	c.Assert(string(serverNode.Value), Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, int64(s.r.leaseID))

	// When
	time.Sleep(3 * time.Second)

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode = res.Kvs[0]
	c.Assert(string(serverNode.Value), Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, int64(s.r.leaseID))
}

// When registry is stopped the backend server record is immediately removed,
// but the backend type record is left intact.
func (s *RegistrySuite) TestHeartbeatStop(c *C) {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode := res.Kvs[0]
	c.Assert(string(serverNode.Value), Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, int64(s.r.leaseID))

	// When
	s.r.Stop()

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/backend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(len(res.Kvs), Equals, 0)
}

func (s *RegistrySuite) TestHeartbeatNetworkTimeout(c *C) {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	c.Assert(string(res.Kvs[0].Value), Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(s.r.leaseID))
	prevLease := s.r.leaseID

	// When
	s.proxy.Disable()

	// Wait 3 seconds
	<-time.After(time.Second * 3)

	s.proxy.Enable()

	// Give time to reconnect
	<-time.After(time.Second * 2)

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	c.Assert(string(res.Kvs[0].Value), Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(s.r.leaseID))
	c.Assert(s.r.leaseID, Not(Equals), prevLease)
}
