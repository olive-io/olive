/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package runner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	urlpkg "net/url"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/client"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/runner/config"
	"github.com/olive-io/olive/runner/delegate"
	delegateGrpc "github.com/olive-io/olive/runner/delegate/service/grpc"
	delegateHttp "github.com/olive-io/olive/runner/delegate/service/http"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/server"
	"github.com/olive-io/olive/runner/storage"
)

const (
	curRevKey = "/watch/revision"
)

type Runner struct {
	genericserver.IEmbedServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.Config

	oct         *client.Client
	clientReady chan struct{}

	bs storage.Storage

	// bpmn process scheduler
	sch *scheduler.Scheduler

	gather *gather.Gather

	serve *http.Server
}

func NewRunner(cfg *config.Config) (*Runner, error) {
	lg := cfg.GetLogger()

	lg.Debug("protected directory: " + cfg.DataDir)

	scfg := &storage.Config{
		Dir:       cfg.DBDir(),
		CacheSize: int64(cfg.CacheSize),
		Logger:    lg,
	}

	if cfg.StorageGCInterval != 0 {
		scfg.GCInterval = cfg.StorageGCInterval
		lg.Info("setting storage gc interval",
			zap.Duration("batch interval", cfg.StorageGCInterval))
	}

	bs, err := storage.NewStorage(scfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	embedServer := genericserver.NewEmbedServer(lg)
	runner := &Runner{
		IEmbedServer: embedServer,
		ctx:          ctx,
		cancel:       cancel,
		cfg:          cfg,

		clientReady: make(chan struct{}, 1),

		bs: bs,
	}

	return runner, nil
}

func (r *Runner) Logger() *zap.Logger {
	return r.cfg.GetLogger()
}

func (r *Runner) Start(stopc <-chan struct{}) error {
	if err := r.start(); err != nil {
		return err
	}
	if err := r.startServer(); err != nil {
		return err
	}

	r.Destroy(r.destroy)
	r.GoAttach(r.buildMonClient)
	r.GoAttach(r.process)

	<-stopc

	return r.stop()
}

func (r *Runner) start() error {
	var err error

	gcfg := &gather.Config{
		Name:        r.cfg.Name,
		HeartbeatMs: r.cfg.HeartbeatMs,
	}
	listenURL := r.cfg.AdvertiseURL
	if listenURL == "" {
		listenURL = r.cfg.ListenURL
	}
	gcfg.ListenURL = listenURL

	dcfg := delegate.NewConfig()
	if err = delegate.Init(dcfg); err != nil {
		return fmt.Errorf("init delegate: %w", err)
	}

	if err = delegate.RegisterDelegate(delegateHttp.New()); err != nil {
		return fmt.Errorf("register http delegate: %w", err)
	}
	if err = delegate.RegisterDelegate(delegateGrpc.New()); err != nil {
		return fmt.Errorf("register gRPC delegate: %w", err)
	}

	r.gather, err = gather.NewGather(r.ctx, gcfg, r.bs)
	if err != nil {
		return fmt.Errorf("create delegate: %w", err)
	}

	r.sch, err = r.startScheduler()
	if err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	return nil
}

func (r *Runner) startScheduler() (*scheduler.Scheduler, error) {
	idGen := r.cfg.IdGenerator()
	scfg := scheduler.NewConfig(r.ctx, r.Logger(), idGen, r.bs)
	sc, err := scheduler.NewScheduler(scfg)
	if err != nil {
		return nil, err
	}

	if err = sc.Start(); err != nil {
		return nil, err
	}

	return sc, nil
}

func (r *Runner) startServer() error {
	lg := r.Logger()

	scheme, ts, err := r.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	r.cfg.ListenURL = scheme + ts.Addr().String()
	handler, err := server.RegisterServer(r.ctx, r.cfg, r.sch, r.gather, r.bs)
	if err != nil {
		return err
	}

	r.serve = &http.Server{
		Handler:        handler,
		MaxHeaderBytes: 1024 * 1024 * 20,
	}

	go func() {
		_ = r.serve.Serve(ts)
	}()

	return nil
}

func (r *Runner) createListener() (string, net.Listener, error) {
	cfg := r.cfg
	lg := r.Logger()
	url, err := urlpkg.Parse(cfg.ListenURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return "http://", listener, nil
}

func (r *Runner) buildMonClient() {
	cfg := r.cfg

	interval := r.cfg.HeartbeatInterval()
	timer := time.NewTicker(interval)
	defer timer.Stop()

LOOP:
	for {
		oct, err := client.New(&cfg.Client)
		if err != nil {
			r.Logger().Error("failed to create client", zap.Error(err))
		} else {
			r.oct = oct

			timeoutCtx, cancel := context.WithTimeout(r.ctx, time.Second*3)
			err = oct.Ping(timeoutCtx)
			cancel()
			if err == nil {
				break LOOP
			}
		}

		select {
		case <-timer.C:
		case <-r.StoppingNotify():
			return
		}
	}

	r.Logger().Info("ready to connect olive-mon cluster")
	close(r.clientReady)
}

func (r *Runner) process() {
	select {
	case <-r.StoppingNotify():
		return
	case <-r.clientReady:
	}

	lg := r.Logger()
	interval := r.cfg.HeartbeatInterval()

	ctx := r.ctx
	runner := r.gather.GetRunner()
	var err error
	runner, err = r.oct.Register(ctx, runner)
	if err != nil {
		lg.Error("register runner to olive-mon", zap.Error(err))
		return
	}
	_ = r.gather.SaveRunner(ctx, runner)

	lg.Info("olive-runner registered",
		zap.Uint64("id", runner.Id),
		zap.String("listen-peer-url", runner.ListenURL),
		zap.Uint64("cpu-total", runner.Cpu),
		zap.String("memory", humanize.IBytes(runner.Memory)),
		zap.String("version", runner.Version))

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	}

	rev := r.getRev()
	if rev != 0 {
		opts = append(opts, clientv3.WithRev(rev))
	}

	prefix := path.Join(api.RunnerTopic, fmt.Sprintf("%d", runner.Id))
	resp, err := r.oct.Get(ctx, prefix, opts...)
	if err != nil {
		lg.Error("get runner topic", zap.Error(err))
	} else {
		for _, kv := range resp.Kvs {
			re, e1 := parseEventKV(kv)
			if e1 == nil {
				r.handleEvent(ctx, re)
			}
			r.setRev(kv.ModRevision)
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {

		ticker.Reset(interval)

		rev = r.getRev()
		wopts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
			clientv3.WithFilterDelete(),
		}
		if rev != 0 {
			wopts = append(wopts, clientv3.WithRev(rev))
		}

		tw := r.oct.Watch(ctx, prefix, wopts...)

	LOOP:
		for {
			select {
			case <-r.StoppingNotify():
				return
			case <-ticker.C:
				stat := r.gather.GetStat()
				err = r.oct.Heartbeat(ctx, stat)
				if err != nil {
					lg.Error("olive-runner update runner stat", zap.Error(err))
				}

			case wch, ok := <-tw:
				if !ok {
					break LOOP
				}
				if werr := wch.Err(); werr != nil {
					break LOOP
				}

				for _, event := range wch.Events {
					if modRev := event.Kv.ModRevision; modRev > rev {
						r.setRev(modRev)
					}

					if event.Type == mvccpb.PUT && event.IsCreate() {
						// deletes topic message
						_, _ = r.oct.Delete(ctx, string(event.Kv.Key))
						re, e1 := parseEventKV(event.Kv)
						if e1 != nil {
							continue
						}

						r.handleEvent(r.ctx, re)
					}
				}
			}
		}
	}
}

func (r *Runner) stop() error {
	r.IEmbedServer.Shutdown()
	err := r.serve.Shutdown(r.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *Runner) getRev() int64 {
	key := curRevKey

	resp, _ := r.bs.Get(r.ctx, key)
	if resp == nil || len(resp.Kvs) == 0 {
		return 0
	}

	value := resp.Kvs[0].Value
	value = value[:8]
	return int64(binary.LittleEndian.Uint64(value))
}

func (r *Runner) setRev(rev int64) {
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(rev))

	key := curRevKey
	_ = r.bs.Put(r.ctx, key, value)
}

func (r *Runner) destroy() {
	r.cancel()

	if err := r.sch.Stop(); err != nil {
		r.Logger().Error("stop scheduler", zap.Error(err))
	}
}
