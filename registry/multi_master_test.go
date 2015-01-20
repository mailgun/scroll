package registry

import (
	"fmt"
	"os"
	"testing"

	"github.com/mailgun/go-etcd/etcd"
	"github.com/mailgun/scroll/vulcan/middleware"

	. "gopkg.in/check.v1"
)

func TestMultiMasterStrategy(t *testing.T) {
	TestingT(t)
}

type MultiMasterSuite struct {
	client   *etcd.Client
	registry *MultiMasterStrategy
}

var _ = Suite(&MultiMasterSuite{})

func (s *MultiMasterSuite) SetUpSuite(c *C) {
	machines := []string{"http://127.0.0.1:4001"}
	s.client = etcd.NewClient(machines)
	s.client.Delete("customkey", true)

	s.registry = NewMultiMasterStrategy("customkey", 15, s.client)
}

func (s *MultiMasterSuite) TestRegisterAppCreatesBackend(c *C) {
	_ = s.registry.RegisterApp("name", "host", 12345)
	backend, err := s.client.Get("customkey/backends/name/backend", false, false)

	c.Assert(err, IsNil)
	c.Assert(backend.Node.Value, Equals, `{"Type":"http"}`)
	c.Assert(backend.Node.TTL, Equals, int64(0))
}

func (s *MultiMasterSuite) TestRegisterAppCreatesServer(c *C) {
	_ = s.registry.RegisterApp("name", "host", 12345)

	host, err := os.Hostname()
	key := fmt.Sprintf("customkey/backends/name/servers/%s_12345", host)
	server, err := s.client.Get(key, false, false)

	c.Assert(err, IsNil)
	c.Assert(server.Node.Value, Equals, `{"URL":"http://host:12345"}`)
	c.Assert(server.Node.TTL, Equals, int64(15))
}

func (s *MultiMasterSuite) TestRegisterHandlerCreatesFrontend(c *C) {
	methods := []string{"PUT"}
	middlewares := []middleware.Middleware{}
	_ = s.registry.RegisterHandler("name", "host", "/path/to/server", methods, middlewares)

	frontend, err := s.client.Get("customkey/frontends/host.put.path.to.server/frontend", false, false)

	c.Assert(err, IsNil)
	c.Assert(frontend.Node.Value, Matches, ".*path\\/to\\/server.*")
	c.Assert(frontend.Node.Value, Matches, ".*PUT.*")
	c.Assert(frontend.Node.TTL, Equals, int64(0))
}

func (s *MultiMasterSuite) TestRegisterHandlerCreatesMiddlewares(c *C) {
	methods := []string{"PUT"}
	middlewares := []middleware.Middleware{}
	middlewares = append(middlewares, middleware.Middleware{Type: "test", ID: "id", Spec: "hi"})
	_ = s.registry.RegisterHandler("name", "host", "/path/to/server", methods, middlewares)

	frontend, err := s.client.Get("customkey/frontends/host.put.path.to.server/middlewares/id", false, false)

	c.Assert(err, IsNil)
	c.Assert(frontend.Node.Value, Matches, `{"Type":"test","Id":"id","Priority":0,"Middleware":"hi"}`)
	c.Assert(frontend.Node.TTL, Equals, int64(0))
}
