package vulcand

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/mailgun/log"
)

const (
	localEtcdProxy = "http://127.0.0.1:2379"
	frontendFmt    = "%s/frontends/%s.%s/frontend"
	middlewareFmt  = "%s/frontends/%s.%s/middlewares/%s"
	backendFmt     = "%s/backends/%s/backend"
	serverFmt      = "%s/backends/%s/servers/%s"

	defaultRegistrationTTL = 30 * time.Second
)

type Config struct {
	Chroot string
	TTL    time.Duration
}

type Registry struct {
	cfg           Config
	etcdKeysAPI   etcd.KeysAPI
	backendSpec   *backendSpec
	frontendSpecs []*frontendSpec
	ctx           context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
}

func NewRegistry(cfg Config, appName, ip string, port int) (*Registry, error) {
	backendSpec, err := newBackendSpec(appName, ip, port)
	if err != nil {
		return nil, fmt.Errorf("Failed to create backend: err=(%s)", err)
	}

	if cfg.TTL <= 0 {
		cfg.TTL = defaultRegistrationTTL
	}

	etcdCfg := etcd.Config{Endpoints: []string{localEtcdProxy}}
	etcdClt, err := etcd.New(etcdCfg)
	if err != nil {
		return nil, err
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		for {
			err := etcdClt.AutoSync(ctx, 10*time.Second)
			if err == context.DeadlineExceeded || err == context.Canceled {
				break
			}
			fmt.Print(err)
		}
	}()
	etcdKeysAPI := etcd.NewKeysAPI(etcdClt)
	c := Registry{
		cfg:         cfg,
		backendSpec: backendSpec,
		etcdKeysAPI: etcdKeysAPI,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
	}
	return &c, nil
}

func (r *Registry) AddFrontend(host, path string, methods []string, middlewares []Middleware) {
	r.frontendSpecs = append(r.frontendSpecs, newFrontendSpec(r.backendSpec.AppName, host, path, methods, middlewares))
}

func (r *Registry) Start() error {
	if err := r.registerBackend(r.backendSpec); err != nil {
		return fmt.Errorf("failed to register backend: err=(%v)", err)
	}
	r.wg.Add(1)
	go r.heartbeat()

	for _, fes := range r.frontendSpecs {
		if err := r.registerFrontend(fes); err != nil {
			r.cancelFunc()
			return fmt.Errorf("failed to register frontend: err=(%v)", err)
		}
	}
	return nil
}

func (r *Registry) Stop() {
	r.cancelFunc()
	r.wg.Wait()
}

func (r *Registry) heartbeat() {
	defer r.wg.Done()
	backendServerKey := fmt.Sprintf(serverFmt, r.cfg.Chroot, r.backendSpec.AppName, r.backendSpec.ID)
	backendServerVal := r.backendSpec.serverSpec()
	tick := time.NewTicker(r.cfg.TTL * 3 / 4)
	for {
		select {
		case <-tick.C:
			if _, err := r.etcdKeysAPI.Set(r.ctx, backendServerKey, "", &etcd.SetOptions{PrevExist: etcd.PrevExist, Refresh: true, TTL: r.cfg.TTL}); err != nil {
				log.Errorf("Server TTL refresh failed: err=(%v)", err)
				if _, err := r.etcdKeysAPI.Set(r.ctx, backendServerKey, backendServerVal, &etcd.SetOptions{TTL: r.cfg.TTL}); err != nil {
					log.Errorf("Server create failed: err=(%v)", err)
				}
			}
		case <-r.ctx.Done():
			_, err := r.etcdKeysAPI.Delete(context.Background(), backendServerKey, nil)
			log.Infof("Heartbeat stopped: err=(%v)", err)
			return
		}
	}
}

func (r *Registry) registerBackend(bes *backendSpec) error {
	betKey := fmt.Sprintf(backendFmt, r.cfg.Chroot, bes.AppName)
	betVal := bes.typeSpec()
	_, err := r.etcdKeysAPI.Set(r.ctx, betKey, betVal, nil)
	if err != nil {
		return err
	}
	besKey := fmt.Sprintf(serverFmt, r.cfg.Chroot, bes.AppName, bes.ID)
	besVar := bes.serverSpec()
	_, err = r.etcdKeysAPI.Set(r.ctx, besKey, besVar, &etcd.SetOptions{TTL: r.cfg.TTL})
	return err
}

func (r *Registry) registerFrontend(fes *frontendSpec) error {
	feKey := fmt.Sprintf(frontendFmt, r.cfg.Chroot, fes.Host, fes.ID)
	feVal := fes.spec()
	_, err := r.etcdKeysAPI.Set(r.ctx, feKey, feVal, nil)
	if err != nil {
		return err
	}
	for i, m := range fes.Middlewares {
		m.Priority = i
		mwKey := fmt.Sprintf(middlewareFmt, r.cfg.Chroot, fes.Host, fes.ID, m.ID)
		mwVal, err := json.Marshal(m)
		if err != nil {
			return err
		}
		_, err = r.etcdKeysAPI.Set(r.ctx, mwKey, string(mwVal), nil)
		if err != nil {
			return err
		}
	}
	return nil
}
