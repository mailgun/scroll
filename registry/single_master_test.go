package registry

import (
	"testing"

	"github.com/mailgun/go-etcd/etcd"
	"github.com/mailgun/scroll/vulcan/middleware"

	. "gopkg.in/check.v1"
)

func TestSingleMaster(t *testing.T) {
	TestingT(t)
}

type SingleMasterSuite struct {
	client   *etcd.Client
	registry *SingleMasterStrategy
}

var _ = Suite(&SingleMasterSuite{})

func (s *SingleMasterSuite) SetUpSuite(c *C) {
	machines := []string{"http://127.0.0.1:4001"}
	s.client = etcd.NewClient(machines)
	s.registry = NewSingleMasterStrategy("customkey", 15, s.client)
}

func (s *SingleMasterSuite) SetUpTest(c *C) {
	s.client.Delete("customkey", true)
}

func (s *SingleMasterSuite) TestRegisterAppCreatesBackend(c *C) {
	_ = s.registry.RegisterApp("name", "host", 12345)
	backend, err := s.client.Get("customkey/backends/name/backend", false, false)

	c.Assert(err, IsNil)
	c.Assert(backend.Node.Value, Equals, `{"Type":"http"}`)
	c.Assert(backend.Node.TTL, Equals, int64(0))
}

func (s *SingleMasterSuite) TestMasterServerRegistration(c *C) {
	_ = s.registry.RegisterApp("name", "host", 12345)

	server, err := s.client.Get("customkey/backends/name/servers/master", false, false)

	c.Assert(err, IsNil)
	c.Assert(server.Node.Value, Equals, `{"URL":"http://host:12345"}`)
	c.Assert(server.Node.TTL, Equals, int64(15))
}

func (s *SingleMasterSuite) TestSlaveServerRegistration(c *C) {
	master := NewSingleMasterStrategy("customkey", 15, s.client)
	master.RegisterApp("name", "master", 12345)
	_ = s.registry.RegisterApp("name", "slave", 67890)

	server, err := s.client.Get("customkey/backends/name/servers/master", false, false)

	c.Assert(err, IsNil)
	c.Assert(server.Node.Value, Equals, `{"URL":"http://master:12345"}`)
}

func (s *SingleMasterSuite) TestSlaveServerBecomesMaster(c *C) {
	// Create a master and slave.
	master := NewSingleMasterStrategy("customkey", 15, s.client)
	_ = master.RegisterApp("name", "master", 12345)
	_ = s.registry.RegisterApp("name", "slave", 67890)

	// Remove the old master and re-register the slave.
	_, err := s.client.Delete("customkey/backends/name/servers/master", false)
	_ = s.registry.RegisterApp("name", "slave", 67890)
	_ = master.RegisterApp("name", "master", 67890)

	server, err := s.client.Get("customkey/backends/name/servers/master", false, false)

	c.Assert(err, IsNil)
	c.Assert(master.IsMaster, Equals, false)
	c.Assert(s.registry.IsMaster, Equals, true)
	c.Assert(server.Node.Value, Equals, `{"URL":"http://slave:67890"}`)
}

func (s *SingleMasterSuite) TestRegisterHandlerCreatesFrontend(c *C) {
	methods := []string{"PUT"}
	middlewares := []middleware.Middleware{}
	_ = s.registry.RegisterHandler("name", "host", "/path/to/server", methods, middlewares)

	frontend, err := s.client.Get("customkey/frontends/host.put.path.to.server/frontend", false, false)

	c.Assert(err, IsNil)
	c.Assert(frontend.Node.Value, Matches, ".*path\\/to\\/server.*")
	c.Assert(frontend.Node.Value, Matches, ".*PUT.*")
	c.Assert(frontend.Node.TTL, Equals, int64(0))
}

func (s *SingleMasterSuite) TestRegisterHandlerCreatesMiddlewares(c *C) {
	methods := []string{"PUT"}
	middlewares := []middleware.Middleware{}
	middlewares = append(middlewares, middleware.Middleware{Type: "test", ID: "id", Spec: "hi"})
	_ = s.registry.RegisterHandler("name", "host", "/path/to/server", methods, middlewares)

	frontend, err := s.client.Get("customkey/frontends/host.put.path.to.server/middlewares/id", false, false)

	c.Assert(err, IsNil)
	c.Assert(frontend.Node.Value, Matches, `{"Type":"test","Id":"id","Priority":0,"Middleware":"hi"}`)
	c.Assert(frontend.Node.TTL, Equals, int64(0))
}
