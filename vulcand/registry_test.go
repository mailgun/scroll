package vulcand

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/Shopify/toxiproxy/client"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/stretchr/testify/suite"
)

const testNamespace = "/test-registry"

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}

type RegistrySuite struct {
	suite.Suite
	client     *etcd.Client
	ctx        context.Context
	cancelFunc context.CancelFunc
	r          *Registry
	cfg        Config
	proxy      *toxiproxy.Proxy
}

func (s *RegistrySuite) SetupSuite() {
	var err error

	// This assumes we used to `docker-compose up` to create the toxiproxy
	tox := toxiproxy.NewClient("localhost:8474")
	s.proxy, err = tox.CreateProxy("etcdv3", "proxy:22379", "etcd:2379")
	s.Require().Nil(err)

	//log.InitWithConfig(log.Config{Name: log.Console})
	//log.SetSeverity(log.SeverityDebug)

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
	s.Require().Nil(err)
}

func (s *RegistrySuite) TearDownSuite() {
	s.proxy.Delete()
}

func (s *RegistrySuite) SetupTest() {
	s.ctx, s.cancelFunc = context.WithTimeout(context.Background(), time.Second*20)
	_, err := s.client.Delete(s.ctx, testNamespace, etcd.WithPrefix())
	s.Require().Nil(err)
	s.r, err = NewRegistry(
		s.cfg,
		"app1",
		"192.168.19.2",
		8000)
	s.Require().Nil(err)
	err = s.r.Start()
	s.Require().Nil(err)

}

func (s *RegistrySuite) TearDownTest() {
	s.r.Stop()
	s.cancelFunc()
}

func (s *RegistrySuite) TestRegisterBackend() {
	bes, err := newBackendSpecWithID("foo", "bar", "example.com", 8000)
	s.Require().Nil(err)

	// When
	err = s.r.registerBackend(bes)

	// Then
	s.Require().Nil(err)

	res, err := s.client.Get(s.ctx, testNamespace+"/backends/bar/backend")
	s.Require().Nil(err)
	s.Equal(string(res.Kvs[0].Value), `{"Type":"http"}`)
	s.Equal(res.Kvs[0].Lease, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/backends/bar/servers/foo")
	s.Require().Nil(err)
	s.Equal(string(res.Kvs[0].Value), `{"URL":"http://example.com:8000"}`)
	s.Equal(res.Kvs[0].Lease, int64(s.r.leaseID))
}

func (s *RegistrySuite) TestRegisterFrontend() {
	m := []Middleware{{Type: "bar", ID: "bazz", Spec: "blah"}}
	fes := newFrontendSpec("foo", "host", "/path/to/server", []string{"GET"}, m)

	// When
	err := s.r.registerFrontend(fes)

	// Then
	s.Require().Nil(err)

	res, err := s.client.Get(s.ctx, testNamespace+"/frontends/host.get.path.to.server/frontend")
	s.Require().Nil(err)
	s.Equal(string(res.Kvs[0].Value), `{"Type":"http","BackendId":"foo","Route":"Host(\"host\") && Method(\"GET\") && Path(\"/path/to/server\")","Settings":{"FailoverPredicate":"(IsNetworkError() || ResponseCode() == 503) && Attempts() <= 2","PassHostHeader":true}}`)
	s.Equal(res.Kvs[0].Lease, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/frontends/host.get.path.to.server/middlewares/bazz")
	s.Require().Nil(err)
	s.Equal(string(res.Kvs[0].Value), `{"Type":"bar","Id":"bazz","Priority":0,"Middleware":"blah"}`)
	s.Equal(res.Kvs[0].Lease, int64(0))
}

func (s *RegistrySuite) TestHeartbeatOnly() {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Equal(len(res.Kvs), 1)
	serverNode := res.Kvs[0]
	s.Equal(string(serverNode.Value), `{"URL":"http://192.168.19.2:8000"}`)
	s.Equal(serverNode.Lease, int64(s.r.leaseID))

	// When
	time.Sleep(3 * time.Second)

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Require().Equal(len(res.Kvs), 1)
	serverNode = res.Kvs[0]
	s.Equal(string(serverNode.Value), `{"URL":"http://192.168.19.2:8000"}`)
	s.Equal(serverNode.Lease, int64(s.r.leaseID))
}

// When registry is stopped the backend server record is immediately removed,
// but the backend type record is left intact.
func (s *RegistrySuite) TestHeartbeatStop() {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Equal(len(res.Kvs), 1)
	serverNode := res.Kvs[0]
	s.Equal(string(serverNode.Value), `{"URL":"http://192.168.19.2:8000"}`)
	s.Equal(serverNode.Lease, int64(s.r.leaseID))

	// When
	s.r.Stop()

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/backend")
	s.Require().Nil(err)
	s.Equal(string(res.Kvs[0].Value), `{"Type":"http"}`)
	s.Equal(res.Kvs[0].Lease, int64(0))

	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Equal(len(res.Kvs), 0)
}

func (s *RegistrySuite) TestHeartbeatNetworkTimeout() {
	res, err := s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Require().Equal(len(res.Kvs), 1)
	s.Equal(string(res.Kvs[0].Value), `{"URL":"http://192.168.19.2:8000"}`)
	s.Equal(res.Kvs[0].Lease, int64(s.r.leaseID))
	prevLease := s.r.leaseID

	// When
	s.proxy.Disable()

	// Wait for disconnect to be discovered
	<-time.After(time.Second * 3)

	s.proxy.Enable()

	// Give time to reconnect
	<-time.After(time.Second * 3)

	// Then
	res, err = s.client.Get(s.ctx, testNamespace+"/backends/app1/servers", etcd.WithPrefix())
	s.Require().Nil(err)
	s.Require().Equal(len(res.Kvs), 1)
	s.Equal(string(res.Kvs[0].Value), `{"URL":"http://192.168.19.2:8000"}`)
	s.Equal(res.Kvs[0].Lease, int64(s.r.leaseID))
	s.NotEqual(s.r.leaseID, prevLease)
}
