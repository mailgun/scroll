package middleware

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/mailgun/scroll/vulcand"
	. "gopkg.in/check.v1"
)

const testNamespace = "/test-middleware"

func TestClient(t *testing.T) {
	TestingT(t)
}

type MiddlewareSuite struct {
	client     *etcd.Client
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *vulcand.Registry
	cfg        vulcand.Config
}

var _ = Suite(&MiddlewareSuite{})

func (s *MiddlewareSuite) SetUpSuite(c *C) {
	var err error

	s.cfg = vulcand.Config{
		Namespace: testNamespace,
		Etcd: &etcd.Config{
			Endpoints: []string{"https://localhost:2379"},
			Username:  "root",
			Password:  "rootpw",
			TLS: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	s.client, err = etcd.New(*s.cfg.Etcd)
	c.Assert(err, IsNil)

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()
	_, err = s.client.Delete(ctx, testNamespace, etcd.WithPrefix())
	c.Assert(err, IsNil)
}

func (s *MiddlewareSuite) SetUpTest(c *C) {
	var err error
	s.ctx, s.cancelFunc = context.WithTimeout(context.Background(), time.Second*5)
	s.r, err = vulcand.NewRegistry(s.cfg, "app1", "192.168.19.2", 8000)
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

	res, err := s.client.Get(s.ctx, testNamespace+"/frontends/mail.gun.get.hello.kitty/frontend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"mail.gun\") && Method(\"GET\") && Path(\"/hello/kitty\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/mail.gun.get.hello.kitty/middlewares/rl1")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"ratelimit","Id":"rl1","Priority":0,"Middleware":{"Variable":"host","Requests":1,"PeriodSeconds":2,"Burst":3}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/mail.gun.get.hello.kitty/middlewares/rw1")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"rewrite","Id":"rw1","Priority":1,"Middleware":{"Regexp":".*","Replacement":"medved","RewriteBody":false,"Redirect":true}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/mailch.imp.put.post.pockemon.go/frontend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"mailch.imp\") && MethodRegexp(\"PUT|POST\") && Path(\"/pockemon/go\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/sendgr.ead.head.hail.ceasar/frontend")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"http","BackendId":"app1","Route":"Host(\"sendgr.ead\") && Method(\"HEAD\") && Path(\"/hail/ceasar\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/sendgr.ead.head.hail.ceasar/middlewares/cb1")
	c.Assert(err, IsNil)
	c.Assert(string(res.Kvs[0].Value), Equals, `{"Type":"cbreaker","Id":"cb1","Priority":0,"Middleware":{"Condition":"con","Fallback":"fall","CheckPeriod":1000000000,"FallbackDuration":1000000,"RecoveryDuration":60000000000,"OnTripped":"trip","OnStandby":"by"}}`)
	c.Assert(res.Kvs[0].Lease, Equals, int64(0))
}
