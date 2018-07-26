package scroll

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/mailgun/holster"
	"github.com/mailgun/scroll/vulcand"
	"github.com/pkg/errors"
	"google.golang.org/grpc/grpclog"
)

const (
	// Suggested result set limit for APIs that may return many entries (e.g. paging).
	DefaultLimit = 100

	// Suggested max allowed result set limit for APIs that may return many entries (e.g. paging).
	MaxLimit = 10000

	// Suggested max allowed amount of entries that batch APIs can accept (e.g. batch uploads).
	MaxBatchSize = 1000

	defaultHTTPReadTimeout  = 10 * time.Second
	defaultHTTPWriteTimeout = 60 * time.Second
	defaultHTTPIdleTimeout  = 60 * time.Second
	localInsecureEndpoint   = "http://127.0.0.1:2379"
	localSecureEndpoint     = "https://127.0.0.1:2379"
	defaultRegistrationTTL  = 30 * time.Second
	defaultNamespace        = "/vulcand"
	pathToCertAuthority     = "/etc/mailgun/ssl/localhost/ca.pem"
)

func applyDefaults(cfg *AppConfig) error {
	var envEndpoint, envUser, envPass, envDebug, endpoint,
		tlsCertFile, tlsKeyFile, tlsCaCertFile string

	for k, v := range map[string]*string{
		"ETCD3_ENDPOINT": &envEndpoint,
		"ETCD3_USER":     &envUser,
		"ETCD3_PASSWORD": &envPass,
		"ETCD3_DEBUG":    &envDebug,
		"ETCD3_TLS_CERT": &tlsCertFile,
		"ETCD3_TLS_KEY":  &tlsKeyFile,
		"ETCD3_CA":       &tlsCaCertFile,
	} {
		*v = os.Getenv(k)
	}

	holster.SetDefault(&cfg.HTTP.ReadTimeout, defaultHTTPReadTimeout)
	holster.SetDefault(&cfg.HTTP.WriteTimeout, defaultHTTPWriteTimeout)
	holster.SetDefault(&cfg.HTTP.IdleTimeout, defaultHTTPIdleTimeout)

	holster.SetDefault(&cfg.Vulcand, &vulcand.Config{})
	holster.SetDefault(&cfg.Vulcand.TTL, defaultRegistrationTTL)
	holster.SetDefault(&cfg.Vulcand.Etcd, &etcd.Config{})

	holster.SetDefault(&endpoint, envEndpoint, localInsecureEndpoint)
	holster.SetDefault(&cfg.Vulcand.Etcd.Endpoints, []string{endpoint})

	holster.SetDefault(&cfg.Vulcand.Namespace, defaultNamespace)
	holster.SetDefault(&cfg.Vulcand.Etcd.Username, envUser)
	holster.SetDefault(&cfg.Vulcand.Etcd.Password, envPass)

	if envDebug != "" {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stderr, os.Stderr, os.Stderr, 4))
	}

	if cfg.Vulcand.Etcd.Username == "" {
		return nil
	}

	if cfg.Vulcand.Etcd.Password == "" {
		return fmt.Errorf("etcd username provided but password is empty")
	}

	// If 'user' and 'pass' supplied assume skip verify TLS config
	holster.SetDefault(&cfg.Vulcand.Etcd.TLS, &tls.Config{InsecureSkipVerify: true})
	holster.SetDefault(&tlsCaCertFile, pathToCertAuthority)

	// If the CA file exists use that
	if _, err := os.Stat(tlsCaCertFile); err == nil {
		var rpool *x509.CertPool = nil
		if pemBytes, err := ioutil.ReadFile(tlsCaCertFile); err == nil {
			rpool = x509.NewCertPool()
			rpool.AppendCertsFromPEM(pemBytes)
		} else {
			return errors.Errorf("while loading cert CA file '%s': %s", tlsCaCertFile, err)
		}
		cfg.Vulcand.Etcd.TLS.RootCAs = rpool
		cfg.Vulcand.Etcd.TLS.InsecureSkipVerify = false
	}

	if tlsCertFile != "" && tlsKeyFile != "" {
		tlsCert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
		if err != nil {
			return errors.Errorf("while loading cert '%s' and key file '%s': %s",
				tlsCertFile, tlsKeyFile, err)
		}
		cfg.Vulcand.Etcd.TLS.Certificates = []tls.Certificate{tlsCert}
	}

	// If we provided the default endpoint, make it a secure endpoint
	if cfg.Vulcand.Etcd.Endpoints[0] == localInsecureEndpoint {
		cfg.Vulcand.Etcd.Endpoints[0] = localSecureEndpoint
	}

	// Ensure the endpoint is https://
	if !strings.HasPrefix(cfg.Vulcand.Etcd.Endpoints[0], "https://") {
		return fmt.Errorf("when connecting to etcd via TLS with credentials " +
			"endpoint must begin with https://")
	}

	return nil
}

func fetchEtcdConfig(cfg *AppConfig) error {
	if cfg.Vulcand == nil || cfg.Vulcand.Etcd == nil {
		return errors.New("a valid etcd.Config{} and vulcand.Config{} config is required")
	}

	client, err := etcd.New(*cfg.Vulcand.Etcd)
	if err != nil {
		return errors.Wrapf(err, "failed to create etcd client for config retrieval, cfg=%v", *cfg.Vulcand.Etcd)
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	key := fmt.Sprintf("/mailgun/configs/%s", cfg.Name)
	resp, err := client.Get(ctx, key)
	if err != nil {
		return errors.Wrapf(err, "while retrieving key '%s'", key)
	}

	if len(resp.Kvs) == 0 {
		return errors.Errorf("config not found while retrieving '%s'", key)
	}

	jsonCfg := JSONConfig{}
	if err := json.Unmarshal(resp.Kvs[0].Value, &jsonCfg); err != nil {
		return errors.Wrap(err, "while parsing json from etcd config")
	}

	// Map the json config to our vulcand config
	cfg.Vulcand.Namespace = jsonCfg.VulcandNamespace
	cfg.PublicAPIHost = jsonCfg.PublicAPIHost
	cfg.PublicAPIURL = jsonCfg.PublicAPIURL
	cfg.ProtectedAPIHost = jsonCfg.ProtectedAPIHost
	cfg.ProtectedAPIURL = jsonCfg.ProtectedAPIURL

	return nil
}
