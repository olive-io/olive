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

	gwrt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	krt "k8s.io/apimachinery/pkg/runtime"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/apis/rpc/runnerpb"
	"github.com/olive-io/olive/client-go"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/runner/delegate"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/server"
	"github.com/olive-io/olive/runner/storage"
	"github.com/olive-io/olive/runner/storage/backend"
)

const (
	curRevKey = "/watch/revision"
)

type Runner struct {
	genericdaemon.IDaemon

	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	oct *clientgo.Client

	be backend.IBackend
	bs *storage.Storage

	// bpmn process scheduler
	sch *scheduler.Scheduler

	gather *gather.Gather

	serve *http.Server
}

func NewRunner(cfg *Config, scheme *krt.Scheme) (*Runner, error) {
	lg := cfg.GetLogger()

	lg.Debug("protected directory: " + cfg.DataDir)
	oct, err := clientgo.New(cfg.clientConfig)
	if err != nil {
		return nil, err
	}

	be, err := newBackend(cfg)
	if err != nil {
		return nil, err
	}
	reflectScheme := apis.FromKrtScheme(scheme)

	bs := storage.New(reflectScheme, be)

	ctx, cancel := context.WithCancel(context.Background())
	embedServer := genericdaemon.NewEmbedDaemon(lg)
	runner := &Runner{
		IDaemon: embedServer,
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,

		oct: oct,
		be:  be,
		bs:  bs,
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
	if err := r.startGRPCServer(); err != nil {
		return err
	}

	r.OnDestroy(r.destroy)
	r.GoAttach(r.process)
	r.GoAttach(r.watching)

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
	if err = delegate.Init(dcfg, r.bs); err != nil {
		return fmt.Errorf("init delegate: %w", err)
	}

	r.gather, err = gather.NewGather(r.ctx, gcfg, r.be)
	if err != nil {
		return fmt.Errorf("create delegate: %w", err)
	}

	r.sch, err = r.startScheduler()
	if err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	return nil
}

func (r *Runner) stop() error {
	r.IDaemon.Shutdown()
	r.serve.Shutdown(r.ctx)
	return nil
}

func (r *Runner) startGRPCServer() error {
	lg := r.Logger()

	scheme, ts, err := r.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	r.cfg.ListenURL = scheme + ts.Addr().String()

	handler, err := r.buildHandler()
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

func (r *Runner) buildHandler() (http.Handler, error) {
	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)

	runnerRPC := server.NewGRPCRunnerServer(r.sch, r.gather, r.bs)
	runnerpb.RegisterRunnerRPCServer(gs, runnerRPC)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	muxOpts := []gwrt.ServeMuxOption{}
	gwmux := gwrt.NewServeMux(muxOpts...)

	if err := runnerpb.RegisterRunnerRPCHandlerServer(r.ctx, gwmux, runnerRPC); err != nil {
		return nil, err
	}

	mux.Handle(
		"/v1/",
		wsproxy.WebsocketProxy(
			gwmux,
			wsproxy.WithRequestMutator(
				// Default to the POST method for streams
				func(_ *http.Request, outgoing *http.Request) *http.Request {
					outgoing.Method = "POST"
					return outgoing
				},
			),
			wsproxy.WithMaxRespBodyBufferSize(0x7fffffff),
		),
	)

	root := http.NewServeMux()
	root.Handle("/", mux)

	handler := genericdaemon.HybridHandler(gs, root)

	return handler, nil
}

func (r *Runner) startScheduler() (*scheduler.Scheduler, error) {
	scfg := scheduler.NewConfig(r.ctx, r.Logger(), r.bs)
	sc, err := scheduler.NewScheduler(scfg)
	if err != nil {
		return nil, err
	}

	if err = sc.Start(); err != nil {
		return nil, err
	}

	return sc, nil
}

func (r *Runner) watching() {
	//ctx, cancel := context.WithCancel(r.ctx)
	//defer cancel()
	//
	//rev := r.getRev()
	//wopts := []clientv3.OpOption{
	//	clientv3.WithPrefix(),
	//	clientv3.WithPrevKV(),
	//	clientv3.WithRev(rev + 1),
	//}
	//wch := r.oct.Watch(ctx, runtime.DefaultRunnerPrefix, wopts...)
	//for {
	//	select {
	//	case <-r.StoppingNotify():
	//		return
	//	case resp := <-wch:
	//		for _, event := range resp.Events {
	//			if event.Kv.ModRevision > rev {
	//				rev = event.Kv.ModRevision
	//				r.setRev(rev)
	//			}
	//			r.processEvent(r.ctx, event)
	//		}
	//	}
	//}
}

func (r *Runner) getRev() int64 {
	key := curRevKey

	rsp, _ := r.be.Get(r.ctx, key)
	if rsp == nil || len(rsp.Kvs) == 0 {
		return 0
	}

	value := rsp.Kvs[0].Value
	value = value[:8]
	return int64(binary.LittleEndian.Uint64(value))
}

func (r *Runner) setRev(rev int64) {
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(rev))

	key := curRevKey
	_ = r.be.Put(r.ctx, key, value)
}

func (r *Runner) destroy() {
	r.cancel()

	if err := r.sch.Stop(); err != nil {
		r.Logger().Error("stop scheduler", zap.Error(err))
	}
}

func (r *Runner) processEvent(ctx context.Context, event *clientv3.Event) {
	//kv := event.Kv
	//key := string(kv.Key)
	//if event.Type == clientv3.EventTypeDelete {
	//	rev := event.Kv.CreateRevision
	//	options := []clientv3.OpOption{clientv3.WithRev(rev)}
	//	rsp, err := r.oct.Get(ctx, string(event.Kv.Key), options...)
	//	if err != nil {
	//		r.Logger().Error("get key-value")
	//	}
	//	if len(rsp.Kvs) > 0 {
	//		event.Kv = rsp.Kvs[0]
	//	}
	//}
	//switch {
	//case strings.HasPrefix(key, runtime.DefaultRunnerRegion):
	//	r.processRegion(ctx, event)
	//case strings.HasPrefix(key, runtime.DefaultRunnerDefinitions):
	//	r.processBpmnDefinition(ctx, event)
	//case strings.HasPrefix(key, runtime.DefaultRunnerProcessInstance):
	//	r.processBpmnProcess(ctx, event)
	//}
}

//func (r *Runner) processRegion(ctx context.Context, event *clientv3.Event) {
//	lg := r.Logger()
//	kv := event.Kv
//	if event.Type == clientv3.EventTypeDelete {
//		return
//	}
//
//	region, match, err := parseRegionKV(kv, r.pr.Id)
//	if err != nil {
//		lg.Error("parse region data", zap.Error(err))
//		return
//	}
//	if !match {
//		return
//	}
//
//	if event.IsCreate() {
//		lg.Info("create region", zap.Stringer("body", region))
//		if err = r.controller.CreateRegion(ctx, region); err != nil {
//			lg.Error("create region", zap.Error(err))
//		}
//		return
//	}
//
//	if event.IsModify() {
//		lg.Info("sync region", zap.Stringer("body", region))
//		if err = r.controller.SyncRegion(ctx, region); err != nil {
//			lg.Error("sync region", zap.Error(err))
//		}
//		return
//	}
//}
//
//func (r *Runner) processBpmnDefinition(ctx context.Context, event *clientv3.Event) {
//	lg := r.Logger()
//	kv := event.Kv
//	if event.Type == clientv3.EventTypeDelete {
//		return
//	}
//
//	definition, match, err := parseDefinitionKV(kv)
//	if err != nil {
//		lg.Error("parse definition data", zap.Error(err))
//		return
//	}
//	if !match {
//		return
//	}
//
//	if err = r.controller.DeployDefinition(ctx, definition); err != nil {
//		lg.Error("definition deploy",
//			zap.String("id", definition.Id),
//			zap.Uint64("version", definition.Version),
//			zap.Error(err))
//		return
//	}
//}
//
//func (r *Runner) processBpmnProcess(ctx context.Context, event *clientv3.Event) {
//	lg := r.Logger()
//	kv := event.Kv
//	if event.Type == clientv3.EventTypeDelete {
//		return
//	}
//
//	process, match, err := parseProcessInstanceKV(kv)
//	if err != nil {
//		lg.Error("parse process data", zap.Error(err))
//		return
//	}
//	if !match {
//		return
//	}
//
//	if err = r.controller.ExecuteDefinition(ctx, process); err != nil {
//		lg.Error("execute definition",
//			zap.String("id", process.DefinitionsId),
//			zap.Uint64("version", process.DefinitionsVersion),
//			zap.Error(err))
//		return
//	}
//
//	commitProcessInstance(ctx, lg, r.oct, process)
//}
