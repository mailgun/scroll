package vulcand

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc/grpclog"
	"github.com/mailgun/holster"
)

const (
	localInsecureEndpoint  = "http://127.0.0.1:2379"
	localSecureEndpoint    = "https://127.0.0.1:2379"
	defaultRegistrationTTL = 30 * time.Second
	defaultChroot          = "/vulcand2"
)

func applyDefaults(cfg *Config) error {
	var envEndpoint, envUser, envPass, envChroot, envDebug, endpoint string

	for k, v := range map[string]*string{
		"ETCD3_ENDPOINT":          &envEndpoint,
		"ETCD3_USER":              &envUser,
		"ETCD3_PASSWORD":          &envPass,
		"ETCD3_VULCAND_NAMESPACE": &envChroot,
		"ETCD3_DEBUG":             &envDebug,
	} {
		*v = os.Getenv(k)
	}

	holster.SetDefault(&cfg.TTL, defaultRegistrationTTL)
	holster.SetDefault(&cfg.Etcd, &etcd.Config{})

	holster.SetDefault(&endpoint, envEndpoint, localInsecureEndpoint)
	holster.SetDefault(&cfg.Etcd.Endpoints, []string{endpoint})

	holster.SetDefault(&cfg.Chroot, envChroot, defaultChroot)
	holster.SetDefault(&cfg.Etcd.Username, envUser)
	holster.SetDefault(&cfg.Etcd.Password, envPass)

	if envDebug != "" {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stderr, os.Stderr, os.Stderr, 4))
	}

	if cfg.Etcd.Username == "" {
		return nil
	}

	if cfg.Etcd.Password == "" {
		return fmt.Errorf("etcd username provided but password is empty")
	}

	// If 'user' and 'pass' supplied assume skip verify TLS config
	holster.SetDefault(&cfg.Etcd.TLS, &tls.Config{ InsecureSkipVerify: true })

	// If we provided the default endpoint, make it a secure endpoint
	if cfg.Etcd.Endpoints[0] == localInsecureEndpoint {
		cfg.Etcd.Endpoints[0] = localSecureEndpoint
	}

	// Ensure the endpoint is https://
	if !strings.HasPrefix(cfg.Etcd.Endpoints[0], "https://") {
		return fmt.Errorf("when connecting to etcd via TLS with credentials " +
			"endpoint must begin with https://")
	}

	return nil
}
