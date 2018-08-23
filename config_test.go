package scroll

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	"os"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/mailgun/scroll/vulcand"
	. "gopkg.in/check.v1"
)

const (
	testNamespace = "/test-config"
)

func TestConfig(t *testing.T) {
	TestingT(t)
}

type ConfigSuite struct{}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) TestFetchEtcdConfig(c *C) {
	var err error

	// This assumes we used to `docker-compose up` to create the etcd node
	cfg := AppConfig{
		PublicAPIHost:    "pub_host",
		PublicAPIURL:     "pub_url",
		ProtectedAPIHost: "prot_host",
		ProtectedAPIURL:  "prot_url",
		Name:             "test-app",

		Vulcand: &vulcand.Config{
			Namespace: testNamespace,
			Etcd: &etcd.Config{
				Endpoints: []string{"https://localhost:2379"},
				Username:  "root",
				Password:  "rootpw",
				TLS: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			TTL: time.Second, // Set a short timeout so we can test keep alive disconnects
		},
	}

	client, err := etcd.New(*cfg.Vulcand.Etcd)
	c.Assert(err, IsNil)

	jsonConfig := JSONConfig{
		PublicAPIHost:    cfg.PublicAPIHost,
		PublicAPIURL:     cfg.PublicAPIURL,
		ProtectedAPIHost: cfg.ProtectedAPIHost,
		ProtectedAPIURL:  cfg.ProtectedAPIURL,
		VulcandNamespace: cfg.Vulcand.Namespace,
	}

	configJson, err := json.Marshal(&jsonConfig)
	c.Assert(err, IsNil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Set an app config in etcd
	_, err = client.Put(ctx, "/mailgun/configs/test/test-app", string(configJson))
	c.Assert(err, IsNil)

	os.Setenv("MG_ENV", "test")

	// fetch the config
	err = fetchEtcdConfig(&cfg)
	c.Assert(err, IsNil)

	c.Assert(cfg.Vulcand.Namespace, Equals, testNamespace)
	c.Assert(cfg.PublicAPIHost, Equals, "pub_host")
	c.Assert(cfg.PublicAPIURL, Equals, "pub_url")
	c.Assert(cfg.ProtectedAPIHost, Equals, "prot_host")
	c.Assert(cfg.ProtectedAPIURL, Equals, "prot_url")
}
