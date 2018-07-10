package vulcand

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/mailgun/log"
	"github.com/pkg/errors"
)

const (
	reconnectInterval = time.Second
	frontendFmt       = "%s/frontends/%s.%s/frontend"
	middlewareFmt     = "%s/frontends/%s.%s/middlewares/%s"
	backendFmt        = "%s/backends/%s/backend"
	serverFmt         = "%s/backends/%s/servers/%s"
)

type Config struct {
	Namespace string
	Etcd      *etcd.Config
	TTL       time.Duration
}

type Registry struct {
	cfg           Config
	client        *etcd.Client
	backendSpec   *backendSpec
	frontendSpecs []*frontendSpec
	ctx           context.Context
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
	leaseID       etcd.LeaseID
	keepAliveChan <-chan *etcd.LeaseKeepAliveResponse
	once          *sync.Once
	done          chan struct{}
}

func NewRegistry(cfg Config, appName, ip string, port int) (*Registry, error) {
	backendSpec, err := newBackendSpec(appName, ip, port)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create backend")
	}

	c := Registry{
		cfg:         cfg,
		backendSpec: backendSpec,
	}
	return &c, nil
}

func (r *Registry) AddFrontend(host, path string, methods []string, middlewares []Middleware) {
	r.frontendSpecs = append(r.frontendSpecs, newFrontendSpec(r.backendSpec.AppName, host, path, methods, middlewares))
}

func (r *Registry) createNewLease() error {
	return nil
}

func (r *Registry) Start() error {
	heartBeatTicker := time.Tick(r.cfg.TTL)
	r.done = make(chan struct{})
	r.once = &sync.Once{}

	// Report any errors the first time we connect
	if err := r.connectAndRegister(); err != nil {
		return err
	}

	const (
		connected = iota + 1
		reconnecting
		alive
	)

	go func() {
		var status int
		r.wg.Add(1)
		for {
			select {
			case <-heartBeatTicker:
				// If we have NOT received a keep alive response during the ticker interval
				// assume we should reconnect and register
				if status != alive {
					for {
						if err := r.connectAndRegister(); err != nil {
							log.Errorf("while reconnecting to etcd: %s", err)
							status = reconnecting
							wait := time.After(reconnectInterval)
							select {
							case <-r.done:
								return
							case <-wait:
								continue
							}
						}
						break
					}
				}
				// This just indicates we reconnected, but haven't received a keep alive response
				status = connected
			case keep := <-r.keepAliveChan:
				if keep != nil {
					log.Debugf("keep alive %+v", keep)
					status = alive
				}
			case <-r.done:
				_, err := r.client.Revoke(context.Background(), r.leaseID)
				log.Infof("lease revoked err=(%v)", err)
				r.wg.Done()
				return
			}
		}
	}()

	return nil
}

func (r *Registry) connectAndRegister() error {
	var err error

	// If we are reconnecting, cancel the previous connections
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	if r.cfg.Etcd == nil {
		return errors.New("a valid *etcd.Config{} is required")
	}

	r.client, err = etcd.New(*r.cfg.Etcd)
	if err != nil {
		return errors.Wrapf(err, "failed to create Etcd client, cfg=%v", *r.cfg.Etcd)
	}
	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	// Grant a new lease for this client instance
	resp, err := r.client.Grant(r.ctx, int64(r.cfg.TTL.Seconds()))
	if err != nil {
		return errors.Wrapf(err, "failed to grant a new lease, cfg=%v", *r.cfg.Etcd)
	}

	// Keep the lease alive for as long as we live
	r.keepAliveChan, err = r.client.KeepAlive(r.ctx, resp.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to start keep alive, cfg=%v", *r.cfg.Etcd)
	}
	r.leaseID = resp.ID

	if err := r.registerBackend(r.backendSpec); err != nil {
		return errors.Wrapf(err, "failed to register backend, %s", r.backendSpec.ID)
	}

	// Write our backend spec config
	key := fmt.Sprintf(serverFmt, r.cfg.Namespace, r.backendSpec.AppName, r.backendSpec.ID)
	_, err = r.client.Put(r.ctx, key, r.backendSpec.serverSpec(), etcd.WithLease(r.leaseID))
	if err != nil {
		return errors.Wrap(err, "failed to write backend spec")
	}

	for _, fes := range r.frontendSpecs {
		if err := r.registerFrontend(fes); err != nil {
			r.cancelFunc()
			return errors.Wrapf(err, "failed to register frontend, %s", fes.ID)
		}
	}
	return nil
}

func (r *Registry) Stop() {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	if r.once != nil {
		r.once.Do(func() { close(r.done) })
	}
	r.wg.Wait()
}

func (r *Registry) registerBackend(bes *backendSpec) error {
	betKey := fmt.Sprintf(backendFmt, r.cfg.Namespace, bes.AppName)
	betVal := bes.typeSpec()
	_, err := r.client.Put(r.ctx, betKey, betVal)
	if err != nil {
		return errors.Wrapf(err, "failed to set backend type, %s", betKey)
	}
	besKey := fmt.Sprintf(serverFmt, r.cfg.Namespace, bes.AppName, bes.ID)
	besVar := bes.serverSpec()
	_, err = r.client.Put(r.ctx, besKey, besVar, etcd.WithLease(r.leaseID))
	return errors.Wrapf(err, "failed to set backend spec, %s", besKey)
}

func (r *Registry) registerFrontend(fes *frontendSpec) error {
	fesKey := fmt.Sprintf(frontendFmt, r.cfg.Namespace, fes.Host, fes.ID)
	fesVal := fes.spec()
	_, err := r.client.Put(r.ctx, fesKey, fesVal)
	if err != nil {
		return errors.Wrapf(err, "failed to set frontend spec, %s", fesKey)
	}
	for i, mw := range fes.Middlewares {
		mw.Priority = i
		mwKey := fmt.Sprintf(middlewareFmt, r.cfg.Namespace, fes.Host, fes.ID, mw.ID)
		mwVal, err := json.Marshal(mw)
		if err != nil {
			return errors.Wrapf(err, "failed to JSON middleware, %v", mw)
		}
		_, err = r.client.Put(r.ctx, mwKey, string(mwVal))
		if err != nil {
			return errors.Wrapf(err, "failed to set middleware, %s", mwKey)
		}
	}
	return nil
}
