package vulcand

import (
	"context"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	. "gopkg.in/check.v1"
)

const testChroot = "/test"

func TestClient(t *testing.T) {
	TestingT(t)
}

type RegistrySuite struct {
	client     *etcd.Client
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *Registry
}

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpSuite(c *C) {
	var err error
	s.client, err = etcd.New(etcd.Config{Endpoints: []string{localEtcdProxy}})
	c.Assert(err, IsNil)
}

func (s *RegistrySuite) SetUpTest(c *C) {
	s.ctx, s.cancelFunc = context.WithTimeout(context.Background(), time.Second*5)
	_, err := s.client.Delete(s.ctx, testChroot, etcd.WithPrefix())
	c.Assert(err, IsNil)
	s.r, err = NewRegistry(Config{Chroot: testChroot}, "app1", "192.168.19.2", 8000)
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

	res, err := s.client.Get(s.ctx, testChroot+"/backends/bar/backend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Kvs[0].Value, Equals, `{"Type":"http"}`)
	c.Assert(res.Kvs[0].Lease, Equals, s.r.leaseID)

	res, err = s.client.Get(s.ctx, testChroot+"/backends/bar/servers/foo", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Kvs[0].Value, Equals, `{"URL":"http://example.com:8000"}`)
	c.Assert(res.Kvs[0], Equals, s.r.leaseID)
}

func (s *RegistrySuite) TestRegisterFrontend(c *C) {
	m := []Middleware{{Type: "bar", ID: "bazz", Spec: "blah"}}
	fes := newFrontendSpec("foo", "host", "/path/to/server", []string{"GET"}, m)

	// When
	err := s.r.registerFrontend(fes)

	// Then
	c.Assert(err, IsNil)

	res, err := s.client.Get(s.ctx, testChroot+"/frontends/host.get.path.to.server/frontend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Kvs[0].Value, Equals, `{"Type":"http","BackendId":"foo","Route":"Host(\"host\") && Method(\"GET\") && Path(\"/path/to/server\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Kvs[0], Equals, s.r.leaseID)

	res, err = s.client.Get(s.ctx, testChroot+"/frontends/host.get.path.to.server/middlewares/bazz", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Kvs[0].Value, Equals, `{"Type":"bar","Id":"bazz","Priority":0,"Middleware":"blah"}`)
	c.Assert(res.Kvs[0], Equals, s.r.leaseID)
}

func (s *RegistrySuite) TestHeartbeat(c *C) {
	s.r.cfg.TTL = time.Second
	err := s.r.Start()
	res, err := s.client.Get(s.ctx, testChroot+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode := res.Kvs[0]
	c.Assert(serverNode.Value, Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, s.r.leaseID)

	// When
	time.Sleep(3 * time.Second)

	// Then
	res, err = s.client.Get(s.ctx, testChroot+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode = res.Kvs[0]
	c.Assert(serverNode.Value, Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, s.r.leaseID)
}

// When registry is stopped the backend server record is immediately removed,
// but the backend type record is left intact.
func (s *RegistrySuite) TestHeartbeatStop(c *C) {
	err := s.r.Start()

	res, err := s.client.Get(s.ctx, testChroot+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Kvs))
	serverNode := res.Kvs[0]
	c.Assert(serverNode.Value, Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.Lease, Equals, s.r.leaseID)

	// When
	s.r.Stop()

	// Then
	res, err = s.client.Get(s.ctx, testChroot+"/backends/app1/backend")
	c.Assert(err, IsNil)
	c.Assert(res.Kvs[0].Value, Equals, `{"Type":"http"}`)
	c.Assert(res.Kvs[0].Lease, Equals, s.r.leaseID)

	res, err = s.client.Get(s.ctx, testChroot+"/backends/app1/servers", etcd.WithPrefix())
	c.Assert(err, IsNil)
	c.Assert(0, Equals, len(res.Kvs))
}
