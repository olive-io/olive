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
	"net"
	"net/http"
	urlpkg "net/url"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/go-logr/zapr"
	"github.com/gofrs/flock"
	gw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/olive-io/bpmn/tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/olive-io/olive/apis"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/client-go"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	dsy "github.com/olive-io/olive/pkg/discovery"
	ort "github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/olive-io/olive/runner/raft"
)

type Runner struct {
	genericdaemon.IDaemon

	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	gLock *flock.Flock

	scheme *krt.Scheme

	oct   *client.Client
	be    backend.IBackend
	serve *http.Server

	controller *raft.Controller
	traces     <-chan tracing.ITrace

	// rw locker for local *monv1.Runner
	prMu sync.RWMutex
	pr   *corev1.Runner
}

func NewRunner(cfg *Config) (*Runner, error) {
	lg := cfg.GetLogger()
	// set klog by zap Logger
	klog.SetLogger(zapr.NewLogger(lg))

	gLock, err := cfg.LockDataDir()
	if err != nil {
		return nil, err
	}
	lg.Debug("protected directory: " + cfg.DataDir)

	oct, err := client.New(cfg.clientConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	be := newBackend(cfg)

	embedDaemon := genericdaemon.NewEmbedDaemon(lg)
	runner := &Runner{
		IDaemon: embedDaemon,
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		gLock:   gLock,
		scheme:  apis.Scheme,
		oct:     oct,
		be:      be,
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

	defer r.be.ForceCommit()
	tx := r.be.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Meta)
	tx.UnsafeCreateBucket(buckets.Region)
	tx.UnsafeCreateBucket(buckets.Key)
	tx.Unlock()
	if err = tx.Commit(); err != nil {
		r.Logger().Error("pebble commit", zap.Error(err))
	}

	runner, err := r.register()
	if err != nil {
		return err
	}
	r.persistRunner(runner)

	r.controller, r.traces, err = r.startRaftController()
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) stop() error {
	r.IDaemon.Shutdown()
	return nil
}

func (r *Runner) startGRPCServer() error {
	lg := r.Logger()

	scheme, ts, err := r.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	r.cfg.ListenClientURL = scheme + ts.Addr().String()

	gs, gwmux, err := r.buildGRPCServer()
	if err != nil {
		return err
	}
	handler := r.buildUserHandler()

	mux := r.createMux(gwmux, handler)
	r.serve = &http.Server{
		Handler:        genericdaemon.GRPCHandlerFunc(gs, mux),
		MaxHeaderBytes: 1024 * 1024 * 20,
	}

	r.GoAttach(func() {
		_ = r.serve.Serve(ts)
	})

	return nil
}

func (r *Runner) createListener() (string, net.Listener, error) {
	cfg := r.cfg
	lg := r.Logger()
	url, err := urlpkg.Parse(cfg.ListenClientURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return url.Scheme + "://", listener, nil
}

func (r *Runner) buildGRPCServer() (*grpc.Server, *gw.ServeMux, error) {
	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)
	mux := gw.NewServeMux()

	runnerGRPC := &runnerImpl{controller: r.controller}
	pb.RegisterRunnerRPCServer(gs, runnerGRPC)
	if err := pb.RegisterRunnerRPCHandlerServer(r.ctx, mux, runnerGRPC); err != nil {
		return nil, nil, err
	}

	return gs, mux, nil
}

func (r *Runner) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())

	return handler
}

func (r *Runner) createMux(gwmux *gw.ServeMux, handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	if gwmux != nil {
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
	}
	if handler != nil {
		mux.Handle("/", handler)
	}
	return mux
}

func (r *Runner) startRaftController() (*raft.Controller, <-chan tracing.ITrace, error) {
	cfg := r.cfg
	lg := r.Logger()

	dopts := []dsy.Option{
		dsy.Prefix(ort.DefaultRunnerDiscoveryNode),
		dsy.SetLogger(lg),
	}
	discovery, err := dsy.NewDiscovery(r.oct.ActiveEtcdClient(), dopts...)
	if err != nil {
		return nil, nil, err
	}

	listenPeerURL, err := urlpkg.Parse(cfg.ListenPeerURL)
	if err != nil {
		return nil, nil, err
	}
	raftAddr := listenPeerURL.Host

	cc := raft.NewConfig()
	cc.DataDir = cfg.RegionDir()
	cc.RaftAddress = raftAddr
	cc.HeartbeatMs = cfg.HeartbeatMs
	cc.RaftRTTMillisecond = cfg.RaftRTTMillisecond
	cc.Logger = lg

	pr := r.getRunner()
	controller, err := raft.NewController(r.ctx, cc, r.be, discovery, pr)
	if err != nil {
		return nil, nil, err
	}

	lg.Info("start raft container")
	if err = controller.Start(r.StoppingNotify()); err != nil {
		return nil, nil, err
	}

	ctx := r.ctx
	prefix := ort.DefaultRunnerRegion
	rev := r.getRev()

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
		clientv3.WithRev(rev),
	}

	rsp, err := r.oct.Get(ctx, prefix, append(options, clientv3.WithMinModRev(rev+1))...)
	if err != nil {
		return nil, nil, err
	}

	regions := make([]*pb.Region, 0)
	definitions := make([]*pb.Definition, 0)
	processes := make([]*pb.ProcessInstance, 0)
	for _, kv := range rsp.Kvs {
		rev = kv.ModRevision
		key := string(kv.Value)
		switch {
		case strings.HasPrefix(key, ort.DefaultRunnerRegion):
			region, match, err := parseRegionKV(kv, uint64(pr.Spec.ID))
			if err != nil || !match {
				continue
			}
			regions = append(regions, region)
		case strings.HasPrefix(key, ort.DefaultRunnerDefinitions):
			definition, match, err := parseDefinitionKV(kv)
			if err != nil || !match {
				continue
			}
			definitions = append(definitions, definition)
		case strings.HasPrefix(key, ort.DefaultRunnerProcessInstance):
			proc, match, err := parseProcessInstanceKV(kv)
			if err != nil || !match {
				continue
			}
			processes = append(processes, proc)
		}
	}
	for _, region := range regions {
		if err = controller.SyncRegion(ctx, region); err != nil {
			return nil, nil, errors.Wrap(err, "sync region")
		}
	}
	for _, definition := range definitions {
		if err = controller.DeployDefinition(ctx, definition); err != nil {
			lg.Error("deploy definition", zap.Error(err))
		}
	}
	for _, process := range processes {
		if err = controller.ExecuteDefinition(ctx, process); err != nil {
			lg.Error("execute definition", zap.Error(err))
		} else {
			commitProcessInstance(ctx, lg, r.oct, process)
		}
	}

	r.setRev(rev)
	return controller, controller.SubscribeTrace(), nil
}

func (r *Runner) watching() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	rev := r.getRev()
	wopts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithPrevKV(),
		clientv3.WithRev(rev + 1),
	}
	wch := r.oct.Watch(ctx, ort.DefaultRunnerPrefix, wopts...)
	for {
		select {
		case <-r.StoppingNotify():
			return
		case resp := <-wch:
			for _, event := range resp.Events {
				if event.Kv.ModRevision > rev {
					rev = event.Kv.ModRevision
					r.setRev(rev)
				}
				r.processEvent(r.ctx, event)
			}
		}
	}
}

func (r *Runner) getRev() int64 {
	key := []byte("cur_rev")
	tx := r.be.ReadTx()
	tx.RLock()
	value, err := tx.UnsafeGet(buckets.Meta, key)
	tx.RUnlock()
	if err != nil || len(value) == 0 {
		return 0
	}
	value = value[:8]
	return int64(binary.LittleEndian.Uint64(value))
}

func (r *Runner) setRev(rev int64) {
	key := []byte("cur_rev")
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(rev))
	tx := r.be.BatchTx()
	tx.Lock()
	_ = tx.UnsafePut(buckets.Meta, key, value)
	tx.Unlock()
	_ = tx.Commit()
}

func (r *Runner) destroy() {
	r.cancel()
	if err := r.gLock.Unlock(); err != nil {
		r.Logger().Error("released "+r.cfg.DataDir, zap.Error(err))
	}
}

func (r *Runner) processEvent(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	key := string(kv.Key)
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
	switch {
	case strings.HasPrefix(key, ort.DefaultRunnerRegion):
		r.processRegion(ctx, event)
	case strings.HasPrefix(key, ort.DefaultRunnerDefinitions):
		r.processBpmnDefinition(ctx, event)
	case strings.HasPrefix(key, ort.DefaultRunnerProcessInstance):
		r.processBpmnProcess(ctx, event)
	}
}

func (r *Runner) processRegion(ctx context.Context, event *clientv3.Event) {
	lg := r.Logger()
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	pr := r.getRunner()
	region, match, err := parseRegionKV(kv, uint64(pr.Spec.ID))
	if err != nil {
		lg.Error("parse region data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if event.IsCreate() {
		lg.Info("create region", zap.Stringer("body", region))
		if err = r.controller.CreateRegion(ctx, region); err != nil {
			lg.Error("create region", zap.Error(err))
		}
		return
	}

	if event.IsModify() {
		lg.Info("sync region", zap.Stringer("body", region))
		if err = r.controller.SyncRegion(ctx, region); err != nil {
			lg.Error("sync region", zap.Error(err))
		}
		return
	}
}

func (r *Runner) processBpmnDefinition(ctx context.Context, event *clientv3.Event) {
	lg := r.Logger()
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	definition, match, err := parseDefinitionKV(kv)
	if err != nil {
		lg.Error("parse definition data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if err = r.controller.DeployDefinition(ctx, definition); err != nil {
		lg.Error("definition deploy",
			zap.String("id", definition.Id),
			zap.Uint64("version", definition.Version),
			zap.Error(err))
		return
	}
}

func (r *Runner) processBpmnProcess(ctx context.Context, event *clientv3.Event) {
	lg := r.Logger()
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	process, match, err := parseProcessInstanceKV(kv)
	if err != nil {
		lg.Error("parse process data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if err = r.controller.ExecuteDefinition(ctx, process); err != nil {
		lg.Error("execute definition",
			zap.String("id", process.DefinitionsId),
			zap.Uint64("version", process.DefinitionsVersion),
			zap.Error(err))
		return
	}

	commitProcessInstance(ctx, lg, r.oct, process)
}
