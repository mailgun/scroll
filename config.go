package scroll

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"os"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/mailgun/holster"
	"github.com/mailgun/holster/etcdutil"
	"github.com/mailgun/scroll/vulcand"
	"github.com/pkg/errors"
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
	defaultRegistrationTTL  = 30 * time.Second
	defaultNamespace        = "/vulcand"
)

func applyDefaults(cfg *AppConfig) error {
	var err error

	holster.SetDefault(&cfg.Vulcand, &vulcand.Config{})
	cfg.Vulcand.Etcd, err = etcdutil.NewEtcdConfig(cfg.Vulcand.Etcd)
	if err != nil {
		return errors.Wrap(err, "while creating new etcd config")
	}

	holster.SetDefault(&cfg.HTTP.ReadTimeout, defaultHTTPReadTimeout)
	holster.SetDefault(&cfg.HTTP.WriteTimeout, defaultHTTPWriteTimeout)
	holster.SetDefault(&cfg.HTTP.IdleTimeout, defaultHTTPIdleTimeout)

	holster.SetDefault(&cfg.Vulcand.TTL, defaultRegistrationTTL)
	holster.SetDefault(&cfg.Vulcand.Etcd, &etcd.Config{})

	holster.SetDefault(&cfg.Vulcand.Namespace, defaultNamespace)

	return nil
}

func fetchEtcdConfig(cfg *AppConfig) error {
	if cfg.Vulcand == nil || cfg.Vulcand.Etcd == nil {
		return errors.New("a valid etcd.Config{} and vulcand.Config{} config is required")
	}

	env := os.Getenv("MG_ENV")
	if env == "" {
		return nil
	}

	client, err := etcd.New(*cfg.Vulcand.Etcd)
	if err != nil {
		return errors.Wrapf(err, "failed to create etcd client for config retrieval, cfg=%v", *cfg.Vulcand.Etcd)
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	key := fmt.Sprintf("/mailgun/configs/%s/%s", env, cfg.Name)
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
