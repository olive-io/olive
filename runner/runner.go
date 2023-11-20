package runner

import (
	"context"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Runner struct {
	Config

	ctx    context.Context
	cancel context.CancelFunc

	ec *clientv3.Client

	stopping chan struct{}
	done     chan struct{}
	stop     chan struct{}

	rmu        sync.RWMutex
	raftGroups map[uint64]*pb.RaftGroup

	wgMu sync.RWMutex
	wg   sync.WaitGroup
}

func NewRunner(cfg Config) (*Runner, error) {
	ec, err := clientv3.New(*cfg.Config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	runner := &Runner{
		Config: cfg,
		ec:     ec,

		ctx:    ctx,
		cancel: cancel,

		stopping: make(chan struct{}, 1),
		done:     make(chan struct{}, 1),
		stop:     make(chan struct{}, 1),
	}

	return runner, nil
}

func (r *Runner) Start() error {
	if err := r.start(); err != nil {
		return err
	}

	r.GoAttach(r.heartbeat)
	r.GoAttach(r.watching)

	return nil
}

func (r *Runner) start() error {
	conn := r.ec.ActiveConnection()
	rc := pb.NewRunnerRPCClient(conn)

	rmsg := &pb.Runner{
		AdvertiseListen: r.AdvertiseListen,
		PeerListen:      r.PeerListen,
	}
	req := &pb.RegistryRunnerRequest{Runner: rmsg}
	rsp, err := rc.RegistryRunner(r.ctx, req)
	if err != nil {
		return err
	}
	rmsg.Id = rsp.Id

	go r.run()
	return nil
}

func (r *Runner) heartbeat() {
	ticker := time.NewTicker(r.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopping:
			return
		case <-ticker.C:

		}
	}
}

func (r *Runner) watching() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	wch := r.ec.Watch(ctx, "")
	for {
		select {
		case <-r.stopping:
			return
		case resp := <-wch:
			for _, event := range resp.Events {
				switch {
				case event.IsModify():
				}
			}
		}
	}
}

// StopNotify returns a channel that receives an empty struct
// when the server is stopped.
func (r *Runner) StopNotify() <-chan struct{} { return r.done }

// StoppingNotify returns a channel that receives an empty struct
// when the server is being stopped.
func (r *Runner) StoppingNotify() <-chan struct{} { return r.stopping }

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (r *Runner) GoAttach(f func()) {
	r.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer r.wgMu.RUnlock()
	select {
	case <-r.stopping:
		r.Logger.Warn("server has stopped; skipping GoAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		f()
	}()
}

func (r *Runner) Stop() {
	select {
	case r.stop <- struct{}{}:
	case <-r.done:
		return
	}
	<-r.done
}

func (r *Runner) run() {
	defer func() {
		r.wgMu.Lock() // block concurrent waitgroup adds in GoAttach while stopping
		close(r.stopping)
		r.wgMu.Unlock()
		r.cancel()

		// wait for goroutines before closing raft so wal stays open
		r.wg.Wait()

		close(r.done)
	}()

	for {
		select {
		case <-r.stop:
			return
		}
	}
}
