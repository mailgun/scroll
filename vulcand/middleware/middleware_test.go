package middleware

import (
	"context"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/mailgun/scroll/vulcand"
	. "gopkg.in/check.v1"
)

const testChroot = "/test"

func TestClient(t *testing.T) {
	TestingT(t)
}

type MiddlewareSuite struct {
	etcdKeyAPI etcd.KeysAPI
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *vulcand.Registry
}

var _ = Suite(&MiddlewareSuite{})

func (s *MiddlewareSuite) SetUpSuite(c *C) {
	etcdClt, err := etcd.New(etcd.Config{Endpoints: []string{"http://127.0.0.1:2379"}})
	c.Assert(err, IsNil)
	s.etcdKeyAPI = etcd.NewKeysAPI(etcdClt)
}

func (s *MiddlewareSuite) SetUpTest(c *C) {
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	s.etcdKeyAPI.Delete(s.ctx, testChroot, &etcd.DeleteOptions{Recursive: true})
	var err error
	s.r, err = vulcand.NewRegistry(vulcand.Config{Chroot: testChroot}, "app1", "192.168.19.2", 8000)
	c.Assert(err, IsNil)
}

func (s *MiddlewareSuite) TearDownTest(c *C) {
	s.r.Stop()
	s.cancelFunc()
}

func (s *MiddlewareSuite) TestMiddlewareRegistration(c *C) {
	s.r.AddFrontend("mail.gun", "/hello/kitty", []string{"get"}, []vulcand.Middleware{
		NewRateLimit(RateLimit{
			Variable:      "host",
			Requests:      1,
			PeriodSeconds: 2,
			Burst:         3}),
		NewRewrite(Rewrite{
			Regexp:      ".*",
			Replacement: "medved",
			Redirect:    true,
			RewriteBody: false,
		}),
	})
	s.r.AddFrontend("mailch.imp", "/pockemon/go", []string{"put", "post"}, nil)
	s.r.AddFrontend("sendgr.ead", "/hail/ceasar", []string{"head"}, []vulcand.Middleware{
		NewCircuitBreaker(CircuitBreaker{
			CheckPeriod:      time.Second,
			Condition:        "con",
			Fallback:         "fall",
			FallbackDuration: time.Millisecond,
			OnStandby:        "by",
			OnTripped:        "trip",
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
