package vulcand

import (
	"context"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/client"
	. "gopkg.in/check.v1"
)

const testChroot = "/test"

func TestClient(t *testing.T) {
	TestingT(t)
}

type RegistrySuite struct {
	etcdKeyAPI etcd.KeysAPI
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *Registry
}

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpSuite(c *C) {
	etcdClt, err := etcd.New(etcd.Config{Endpoints: []string{localEtcdProxy}})
	c.Assert(err, IsNil)
	s.etcdKeyAPI = etcd.NewKeysAPI(etcdClt)
}

func (s *RegistrySuite) SetUpTest(c *C) {
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	s.etcdKeyAPI.Delete(s.ctx, testChroot, &etcd.DeleteOptions{Recursive: true})
	var err error
	s.r, err = NewRegistry(Config{Chroot: testChroot}, "app1", "192.168.19.2", 8000)
	c.Assert(err, IsNil)
}

func (s *RegistrySuite) TearDownTest(c *C) {
	s.r.Stop()
	s.cancelFunc()
}

func (s *RegistrySuite) TestRegisterBackendType(c *C) {
	bes, err := newBackendSpecWithID("foo", "bar", "example.com", 8000)
	c.Assert(err, IsNil)

	// When
	err = s.r.registerBackendType(bes)

	// Then
	c.Assert(err, IsNil)
	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/bar/backend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http"}`)
	c.Assert(res.Node.TTL, Equals, int64(0))
}

func (s *RegistrySuite) TestRegisterBackendServer(c *C) {
	bes, err := newBackendSpecWithID("foo", "bar", "example.com", 8000)
	c.Assert(err, IsNil)

	// When
	err = s.r.registerBackendServer(bes, 15*time.Second)

	// Then
	c.Assert(err, IsNil)
	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/bar/servers/foo", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"URL":"http://example.com:8000"}`)
	c.Assert(res.Node.TTL, Equals, int64(15))
}

func (s *RegistrySuite) TestRegisterBackendServerAgain(c *C) {
	bes, err := newBackendSpecWithID("foo", "bar", "example.com", 8000)
	c.Assert(err, IsNil)
	err = s.r.registerBackendServer(bes, 15*time.Second)
	c.Assert(err, IsNil)

	// When
	err = s.r.registerBackendServer(bes, 15*time.Second)

	// Then
	c.Assert(err, IsNil)
	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/bar/servers/foo", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"URL":"http://example.com:8000"}`)
	c.Assert(res.Node.TTL, Equals, int64(15))
}

func (s *RegistrySuite) TestRegisterFrontend(c *C) {
	m := []Middleware{{Type: "bar", ID: "bazz", Spec: "blah"}}
	fes := newFrontendSpec("foo", "host", "/path/to/server", []string{"GET"}, m)

	// When
	err := s.r.registerFrontend(fes)

	// Then
	c.Assert(err, IsNil)

	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/host.get.path.to.server/frontend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http","BackendId":"foo","Route":"Host(\"host\") && Method(\"GET\") && Path(\"/path/to/server\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/host.get.path.to.server/middlewares/bazz", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"bar","Id":"bazz","Priority":0,"Middleware":"blah"}`)
	c.Assert(res.Node.TTL, Equals, int64(0))
}

// When registry is stopped the backend server record is immediately removed,
// but the backend type record is left intact.
func (s *RegistrySuite) TestHeartbeatStop(c *C) {
	err := s.r.Start()

	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/app1/servers", &etcd.GetOptions{Recursive: true})
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Node.Nodes))
	serverNode := res.Node.Nodes[0]
	c.Assert(serverNode.Value, Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.TTL, Equals, int64(defaultRegistrationTTL/time.Second))

	// When
	s.r.Stop()

	// Then
	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/app1/backend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http"}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/backends/app1/servers", &etcd.GetOptions{Recursive: true})
	c.Assert(err, IsNil)
	c.Assert(0, Equals, len(res.Node.Nodes))
}
