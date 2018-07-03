package vulcand

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc/grpclog"
)

const (
	localInsecureEndpoint  = "http://127.0.0.1:2379"
	localSecureEndpoint    = "https://127.0.0.1:2379"
	defaultRegistrationTTL = 30 * time.Second
	defaultChroot          = "/vulcand2"
)

func applyDefaults(cfg *Config) error {
	var endpoint, user, pass, namespace, debug string

	if cfg.TTL.Seconds() <= 0 {
		cfg.TTL = defaultRegistrationTTL
	}

	for k, v := range map[string]*string{
		"ETCD3_ENDPOINT":          &endpoint,
		"ETCD3_USER":              &user,
		"ETCD3_PASSWORD":          &pass,
		"ETCD3_VULCAND_NAMESPACE": &namespace,
		"ETCD3_DEBUG":             &debug,
	} {
		*v = os.Getenv(k)
	}

	if cfg.Etcd == nil {
		cfg.Etcd = &etcd.Config{}
	}

	// If no endpoint provided, use default insecure
	if endpoint == "" && len(cfg.Etcd.Endpoints) == 0 {
		cfg.Etcd.Endpoints = []string{localInsecureEndpoint}
	} else {
		if len(cfg.Etcd.Endpoints) == 0 {
			cfg.Etcd.Endpoints = []string{endpoint}
		}
	}

	if namespace != "" && cfg.Chroot == "" {
		cfg.Chroot = namespace
	} else {
		if cfg.Chroot == "" {
			cfg.Chroot = defaultChroot
		}
	}

	if debug != "" {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stderr, os.Stderr, os.Stderr, 4))
	}

	if user == "" && cfg.Etcd.Username == "" {
		return nil
	}

	if pass == "" && cfg.Etcd.Password == "" {
		return fmt.Errorf("'ETCD3_USER' provided but missing 'ETCD3_PASSWORD'")
	}

	// If 'user' and 'pass' supplied assume TLS config
	if user != "" {
		cfg.Etcd.Username = user
	}

	if pass != "" {
		cfg.Etcd.Password = pass
	}

	if cfg.Etcd.TLS == nil {
		cfg.Etcd.TLS = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		cfg.Etcd.TLS.InsecureSkipVerify = true
	}

	if len(cfg.Etcd.Endpoints) != 0 {
		// If we provided the default endpoint, make it a secure endpoint
		if cfg.Etcd.Endpoints[0] == localInsecureEndpoint {
			cfg.Etcd.Endpoints[0] = localSecureEndpoint
		} else {
			// Ensure the endpoint is https://
			if !strings.HasPrefix(cfg.Etcd.Endpoints[0], "https://") {
				return fmt.Errorf("when connecting to etcd via TLS with credentials " +
					"endpoint must begin with https://")
			}
		}
	} else {
		cfg.Etcd.Endpoints = []string{localSecureEndpoint}
	}

	return nil
}
