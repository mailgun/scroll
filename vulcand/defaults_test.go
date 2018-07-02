package vulcand

import (
	"os"

	. "gopkg.in/check.v1"
)

type DefaultSuite struct{}

var _ = Suite(&DefaultSuite{})

func (s *DefaultSuite) TearDownTest(c *C) {
	unStashEnv("ETCD3_USER")
	unStashEnv("ETCD3_PASSWORD")
	unStashEnv("ETCD3_ENDPOINT")
	unStashEnv("ETCD3_VULCAND_NAMESPACE")
}

func (s *DefaultSuite) TestApplyDefault(c *C) {
	stashEnv("ETCD3_USER", "")
	stashEnv("ETCD3_ENDPOINT", "")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg Config
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Chroot, Equals, defaultChroot)
	c.Assert(cfg.TTL, Equals, defaultRegistrationTTL)
	c.Assert(cfg.Etcd.Endpoints[0], Equals, localInsecureEndpoint)
	c.Assert(cfg.Etcd.TLS, IsNil)
}

func (s *DefaultSuite) TestApplyDefaultWithCreds(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg Config
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Chroot, Equals, defaultChroot)
	c.Assert(cfg.TTL, Equals, defaultRegistrationTTL)
	c.Assert(cfg.Etcd.Endpoints[0], Equals, localSecureEndpoint)

	c.Assert(cfg.Etcd.TLS, NotNil)
	c.Assert(cfg.Etcd.TLS.InsecureSkipVerify, Equals, true)

	c.Assert(cfg.Etcd.Username, Equals, "user")
	c.Assert(cfg.Etcd.Password, Equals, "pass")
}

func (s *DefaultSuite) TestApplyDefaultNamespace(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "/mytest")

	var cfg Config
	err := applyDefaults(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Chroot, Equals, "/mytest")
}

func (s *DefaultSuite) TestApplyDefaultWithCredsWrongScheme(c *C) {
	stashEnv("ETCD3_USER", "user")
	stashEnv("ETCD3_PASSWORD", "pass")
	stashEnv("ETCD3_ENDPOINT", "http://localhost:2379")
	stashEnv("ETCD3_VULCAND_NAMESPACE", "")

	var cfg Config
	err := applyDefaults(&cfg)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals,
		"when connecting to etcd via TLS with credentials endpoint must begin with https://")
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
