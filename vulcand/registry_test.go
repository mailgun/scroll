package vulcand

import (
	"context"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/mailgun/scroll/vulcand/middleware"
	. "gopkg.in/check.v1"
)

const (
	testChroot = "/test"
)

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
	m := []middleware.T{{Type: "bar", ID: "bazz", Spec: "blah"}}
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

func (s *RegistrySuite) TestHeartbeatStart(c *C) {
	s.r.AddFrontend("mail.gun", "/hello/kitty", []string{"get"}, []middleware.T{
		middleware.NewRateLimit(middleware.RateLimit{
			Variable: "host",
			Requests: 1,
			PeriodSeconds: 2,
			Burst: 3}),
		middleware.NewRewrite(middleware.Rewrite{
			Regexp: ".*",
			Replacement: "medved",
			Redirect: true,
			RewriteBody: false,
		}),
	})
	s.r.AddFrontend("mailch.imp", "/pockemon/go", []string{"put","post"}, nil)
	s.r.AddFrontend("sendgr.ead", "/hail/ceasar", []string{"head"}, []middleware.T{
		middleware.NewCircuitBreaker(middleware.CircuitBreaker{
			CheckPeriod: time.Second,
			Condition: "con",
			Fallback: "fall",
			FallbackDuration: time.Millisecond,
			OnStandby: "by",
			OnTripped: "trip",
			RecoveryDuration: time.Minute,
		}),
	})

	// When
	err := s.r.Start()

	// Then
	c.Assert(err, IsNil)

	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/mail.gun.get.hello.kitty/frontend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"mail.gun\") && Method(\"get\") && Path(\"/hello/kitty\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/mail.gun.get.hello.kitty/middlewares/rl1", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"ratelimit","Id":"rl1","Priority":0,"Middleware":{"Variable":"host","Requests":1,"PeriodSeconds":2,"Burst":3}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/mail.gun.get.hello.kitty/middlewares/rw1", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"rewrite","Id":"rw1","Priority":1,"Middleware":{"Regexp":".*","Replacement":"medved","RewriteBody":false,"Redirect":true}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/mailch.imp.put.post.pockemon.go/frontend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"mailch.imp\") && MethodRegexp(\"put|post\") && Path(\"/pockemon/go\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/sendgr.ead.head.hail.ceasar/frontend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"sendgr.ead\") && Method(\"head\") && Path(\"/hail/ceasar\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot+"/frontends/sendgr.ead.head.hail.ceasar/middlewares/cb1", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"cbreaker","Id":"cb1","Priority":0,"Middleware":{"Condition":"con","Fallback":"fall","CheckPeriod":1000000000,"FallbackDuration":1000000,"RecoveryDuration":60000000000,"OnTripped":"trip","OnStandby":"by"}}`)
	c.Assert(res.Node.TTL, Equals, int64(0))
}

// When registry is stopped the backend server record is immediately removed,
// but the backend type record is left intact.
func (s *RegistrySuite) TestHeartbeatStop(c *C) {
	err := s.r.Start()

	res, err := s.etcdKeyAPI.Get(s.ctx, testChroot + "/backends/app1/servers", &etcd.GetOptions{Recursive: true})
	c.Assert(err, IsNil)
	c.Assert(1, Equals, len(res.Node.Nodes))
	serverNode := res.Node.Nodes[0]
	c.Assert(serverNode.Value, Equals, `{"URL":"http://192.168.19.2:8000"}`)
	c.Assert(serverNode.TTL, Equals, int64(defaultRegistrationTTL / time.Second))

	// When
	s.r.Stop()

	// Then
	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot + "/backends/app1/backend", nil)
	c.Assert(err, IsNil)
	c.Assert(res.Node.Value, Equals, `{"Type":"http"}`)
	c.Assert(res.Node.TTL, Equals, int64(0))

	res, err = s.etcdKeyAPI.Get(s.ctx, testChroot + "/backends/app1/servers", &etcd.GetOptions{Recursive: true})
	c.Assert(err, IsNil)
	c.Assert(0, Equals, len(res.Node.Nodes))
}
