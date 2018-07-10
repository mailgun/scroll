package scroll

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"testing"
	"time"

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

func (s *ConfigSuite) TearDownTest(c *C) {
	unStashEnv("ETCD3_USER")
	unStashEnv("ETCD3_PASSWORD")
	unStashEnv("ETCD3_ENDPOINT")
	unStashEnv("ETCD3_VULCAND_NAMESPACE")
}

func (s *ConfigSuite) TestApplyDefault(c *C) {
	stashEnv("ETCD3_USER", "")
	stashEnv("ETCD3_ENDPOINT", "")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg AppConfig
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Vulcand.Namespace, Equals, defaultNamespace)
	c.Assert(cfg.Vulcand.TTL, Equals, defaultRegistrationTTL)
	c.Assert(cfg.Vulcand.Etcd.Endpoints[0], Equals, localInsecureEndpoint)
	c.Assert(cfg.Vulcand.Etcd.TLS, IsNil)
}

func (s *ConfigSuite) TestApplyDefaultWithCreds(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg AppConfig
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Vulcand.Namespace, Equals, defaultNamespace)
	c.Assert(cfg.Vulcand.TTL, Equals, defaultRegistrationTTL)
	c.Assert(cfg.Vulcand.Etcd.Endpoints[0], Equals, localSecureEndpoint)

	c.Assert(cfg.Vulcand.Etcd.TLS, NotNil)
	c.Assert(cfg.Vulcand.Etcd.TLS.InsecureSkipVerify, Equals, true)

	c.Assert(cfg.Vulcand.Etcd.Username, Equals, "user")
	c.Assert(cfg.Vulcand.Etcd.Password, Equals, "pass")
}

func (s *ConfigSuite) TestApplyDefaultPreferSetValue(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "http://example.com")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "/bar")

	cfg := AppConfig{
		Vulcand: &vulcand.Config{
			Namespace: "/foo",
			TTL:       time.Second * 10,
			Etcd: &etcd.Config{
				Endpoints: []string{"https://foo.bar"},
				TLS: &tls.Config{
					InsecureSkipVerify: false,
				},
				Username: "kit",
				Password: "kat",
			},
		},
	}

	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Vulcand.Namespace, Equals, "/foo")
	c.Assert(cfg.Vulcand.TTL, Equals, time.Second*10)
	c.Assert(cfg.Vulcand.Etcd.Endpoints[0], Equals, "https://foo.bar")

	c.Assert(cfg.Vulcand.Etcd.TLS, NotNil)
	c.Assert(cfg.Vulcand.Etcd.TLS.InsecureSkipVerify, Equals, false)

	c.Assert(cfg.Vulcand.Etcd.Username, Equals, "kit")
	c.Assert(cfg.Vulcand.Etcd.Password, Equals, "kat")
}

func (s *ConfigSuite) TestApplyDefaultNamespace(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "")

	var cfg AppConfig
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Vulcand.Namespace, Equals, "/vulcand")
}

func (s *ConfigSuite) TestApplyDefaultWithCredsWrongScheme(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "http://localhost:2379")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg AppConfig
	err := applyDefaults(&cfg)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals,
		"when connecting to etcd via TLS with credentials endpoint must begin with https://")
}

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
	_, err = client.Put(ctx, "/mailgun/configs/test-app", string(configJson))
	c.Assert(err, IsNil)

	// fetch the config
	err = fetchEtcdConfig(&cfg)
	c.Assert(err, IsNil)

	c.Assert(cfg.Vulcand.Namespace, Equals, testNamespace)
	c.Assert(cfg.PublicAPIHost, Equals, "pub_host")
	c.Assert(cfg.PublicAPIURL, Equals, "pub_url")
	c.Assert(cfg.ProtectedAPIHost, Equals, "prot_host")
	c.Assert(cfg.ProtectedAPIURL, Equals, "prot_url")
}

var stash map[string]string

func stashEnv(name, value string) {
	if stash == nil {
		stash = make(map[string]string)
	}
	stash[name] = os.Getenv(name)
	os.Setenv(name, value)
}

func unStashEnv(name string) {
	if stash == nil {
		return
	}
	value, ok := stash[name]
	if ok {
		os.Setenv(name, value)
	}
}
