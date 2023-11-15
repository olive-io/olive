package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	dbc "github.com/lni/dragonboat/v4/config"
	sm "github.com/lni/dragonboat/v4/statemachine"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server/auth"
	"github.com/olive-io/olive/server/cindex"
	"github.com/olive-io/olive/server/config"
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/membership"
	"github.com/olive-io/olive/server/mvcc"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

var (
	lastAppliedIndex = []byte("last_applied_index")
)

// RaftStatusGetter represents olive server and Raft progress.
type RaftStatusGetter interface {
	ShardID() uint64
	NodeID() uint64
	Leader() uint64
	AppliedIndex() uint64
	CommittedIndex() uint64
	Term() uint64
}

type Replica struct {
	*config.ServerConfig

	SCfg config.ShardConfig

	s *OliveServer

	lg *zap.Logger

	id     uint64
	nodeID uint64

	msmu    sync.RWMutex
	members map[uint64]string

	cluster *membership.RaftCluster

	isLearner bool

	ctx    context.Context
	cancel context.CancelFunc

	w wait.Wait

	kv     mvcc.IWatchableKV
	lessor lease.ILessor

	bemu      sync.Mutex
	be        backend.IBackend
	authStore auth.AuthStore

	apply     applier
	applyWait wait.WaitTime

	consistIndex cindex.IConsistentIndexer
	reqIDGen     *idutil.Generator

	stopping chan struct{}
	done     chan struct{}
	changec  chan struct{}

	replicaRequestC chan *replicaRequest

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the goroutines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup

	appliedIndex   uint64 // must use atomic operations to access; keep 64-bit aligned.
	committedIndex uint64 // must use atomic operations to access; keep 64-bit aligned.
	term           uint64 // must use atomic operations to access; keep 64-bit aligned.
	lead           uint64 // must use atomic operations to access; keep 64-bit aligned.
}

func (s *OliveServer) NewReplica(sc config.ShardConfig) (ra *Replica, members map[uint64]string, join bool, rc dbc.Config, err error) {
	cfg := s.ServerConfig
	lg := s.Logger()

	rc = dbc.Config{
		ShardID:             sc.ShardID,
		CheckQuorum:         true,
		PreVote:             s.PreVote,
		ElectionRTT:         s.ElectionTTL,
		HeartbeatRTT:        s.HeartBeatTTL,
		SnapshotEntries:     10,
		CompactionOverhead:  5,
		OrderedConfigChange: true,
		WaitReady:           true,
	}

	members = map[uint64]string{}
	for key, urlText := range sc.PeerURLs {
		URL, e1 := url.Parse(urlText.String())
		if e1 != nil {
			err = fmt.Errorf("invalid url: %v", urlText.String())
			return
		}

		id, _ := strconv.ParseUint(key, 10, 64)
		if id == 0 {
			id = GenHash([]byte(key))
		}
		replicaID := id
		peerAddress := URL.Host
		if peerAddress == s.ListenerPeerAddress {
			rc.ReplicaID = replicaID
		}
		members[replicaID] = peerAddress
	}

	if err = rc.Validate(); err != nil {
		return
	}

	if len(members) == 0 || rc.IsNonVoting || rc.IsWitness {
		join = true
	}

	shardID := rc.ShardID
	nodeID := rc.ReplicaID

	if cfg.MaxRequestBytes > recommendedMaxRequestBytes {
		lg.Warn(
			"exceeded recommended request limit",
			zap.Uint64("max-request-bytes", cfg.MaxRequestBytes),
			zap.String("max-request-size", humanize.Bytes(cfg.MaxRequestBytes)),
			zap.Int("recommended-request-bytes", recommendedMaxRequestBytes),
			zap.String("recommended-request-size", recommendedMaxRequestBytesString),
		)
	}

	be := openBackend(lg, shardID, nodeID, *cfg, nil)

	var cl *membership.RaftCluster
	cl, err = membership.NewClusterFromURLsMap(lg, sc.Token, sc.PeerURLs)
	if err != nil {
		return
	}
	cl.SetBackend(be)
	cl.SetID(shardID, nodeID)

	ci := cindex.NewConsistentIndex(be, shardID, nodeID)

	heartbeat := time.Millisecond * time.Duration(cfg.RTTMillisecond) * time.Duration(cfg.HeartBeatTTL)
	minTTL := (3 * time.Millisecond * time.Duration(cfg.RTTMillisecond) * time.Duration(cfg.ElectionTTL)) / 2 * heartbeat

	// always recover lessor before kv. When we recover the mvcc.KV it will reattach keys to its leases.
	// If we recover mvcc.KV first, it will attach the keys to the wrong lessor before it recovers.
	lessor := lease.NewLessor(lg, be, cl, lease.LessorConfig{
		MinLeaseTTL:                int64(math.Ceil(minTTL.Seconds())),
		CheckpointInterval:         cfg.LeaseCheckpointInterval,
		CheckpointPersist:          cfg.LeaseCheckpointPersist,
		ExpiredLeasesRetryInterval: cfg.ReqTimeout(),
	})

	kv := mvcc.New(lg, be, lessor, mvcc.StoreConfig{CompactionBatchLimit: cfg.CompactionBatchLimit})

	ctx, cancel := context.WithCancel(s.ctx)
	ra = &Replica{
		ServerConfig: cfg,
		SCfg:         sc,

		s: s,

		lg: s.Logger(),

		id:     shardID,
		nodeID: nodeID,

		members: members,
		cluster: cl,

		ctx:    ctx,
		cancel: cancel,

		kv:     kv,
		lessor: lessor,
		be:     be,

		consistIndex: ci,

		reqIDGen: idutil.NewGenerator(uint16(nodeID), time.Now()),

		stopping: make(chan struct{}, 1),
		done:     make(chan struct{}),
		changec:  make(chan struct{}, 5),

		replicaRequestC: s.replicaRequestC,
	}

	ra.w = wait.New()
	ra.applyWait = wait.NewTimeList()
	ra.isLearner = rc.IsNonVoting

	tp, err := auth.NewTokenProvider(lg, cfg.AuthToken,
		func(index uint64) <-chan struct{} {
			return ra.applyWait.Wait(index)
		},
		time.Duration(cfg.TokenTTL)*time.Second,
	)
	if err != nil {
		return
	}

	ra.authStore = auth.NewAuthStore(lg, be, tp, int(cfg.BcryptCost))
	ra.apply = ra.newApplier()

	return
}

func (ra *Replica) NewDiskStateMachine(shardID, nodeID uint64) sm.IOnDiskStateMachine {

	ra.s.rmu.Lock()
	ra.s.replicas[shardID] = ra
	ra.s.rmu.Unlock()

	ra.done = make(chan struct{}, 1)

	go ra.run()

	go func() {
		select {
		case <-ra.stopping:
			ra.s.rmu.Lock()
			delete(ra.s.replicas, shardID)
			ra.s.rmu.Unlock()
		}
	}()

	return ra
}

func (ra *Replica) run() {
	defer close(ra.done)
	for {
		select {
		case <-ra.stopping:
			return
		case <-ra.changec:
			// TODO: trigger raft group change
		default:

		}
	}
}

func (ra *Replica) Open(done <-chan struct{}) (uint64, error) {
	ctx, cancel := context.WithCancel(ra.ctx)
	defer cancel()
	r := &pb.RangeRequest{
		Key:   lastAppliedIndex,
		Limit: 1,
	}
	rsp, err := ra.apply.Range(ctx, nil, r)
	if err != nil {
		return 0, err
	}
	if len(rsp.Kvs) == 0 {
		return 0, err
	}

	index := binary.LittleEndian.Uint64(rsp.Kvs[0].Value)
	ra.setAppliedIndex(index)
	return index, nil
}

func (ra *Replica) Update(entries []sm.Entry) ([]sm.Entry, error) {
	if length := len(entries); length > 0 {
		ra.setCommittedIndex(entries[length-1].Index)
	}

	if entries[0].Index < ra.getAppliedIndex() {
		return entries, nil
	}

	last := 0
	for i := range entries {
		entry := entries[i]
		ra.applyEntryNormal(&entry)
		ra.setAppliedIndex(entry.Index)
		entry.Result = sm.Result{Value: uint64(len(entry.Cmd))}
		last = i
	}

	lastIndex := entries[last].Index
	ra.applyWait.Trigger(lastIndex)
	ctx, cancel := context.WithCancel(ra.ctx)
	defer cancel()
	r := &pb.PutRequest{Key: lastAppliedIndex}
	r.Value = make([]byte, 8)
	binary.LittleEndian.PutUint64(r.Value, lastIndex)
	_, _, err := ra.apply.Put(ctx, nil, r)
	if err != nil {
		return entries, err
	}

	return entries, nil
}

func (ra *Replica) Lookup(query interface{}) (interface{}, error) {
	ctx, cancel := context.WithCancel(ra.ctx)
	defer cancel()

	r := query.(*pb.RangeRequest)
	rsp, err := ra.apply.Range(ctx, nil, r)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func (ra *Replica) Sync() error {
	return nil
}

func (ra *Replica) PrepareSnapshot() (interface{}, error) {
	snapshot := ra.be.Snapshot()
	return snapshot, nil
}

func (ra *Replica) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	snapshot := ctx.(backend.ISnapshot)
	_, err := snapshot.WriteTo(writer)
	if err != nil {
		return err
	}

	return nil
}

func (ra *Replica) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	err := ra.be.Recover(reader)
	if err != nil {
		return err
	}

	return nil
}

func (ra *Replica) parseProposeCtxErr(err error, start time.Time) error {
	switch err {
	case context.Canceled:
		return ErrCanceled

	case context.DeadlineExceeded:
		//s.leadTimeMu.RLock()
		//curLeadElected := s.leadElectedTime
		//s.leadTimeMu.RUnlock()
		//prevLeadLost := curLeadElected.Add(-2 * time.Duration(s.Cfg.ElectionTicks) * time.Duration(s.Cfg.TickMs) * time.Millisecond)
		//if start.After(prevLeadLost) && start.Before(curLeadElected) {
		//	return ErrTimeoutDueToLeaderFail
		//}
		//lead := types.ID(s.getLead())
		//switch lead {
		//case types.ID(raft.None):
		//	// TODO: return error to specify it happens because the cluster does not have leader now
		//case s.ID():
		//	if !isConnectedToQuorumSince(s.r.transport, start, s.ID(), s.cluster.Members()) {
		//		return errs.ErrTimeoutDueToConnectionLost
		//	}
		//default:
		//	if !isConnectedSince(s.r.transport, start, lead) {
		//		return errs.ErrTimeoutDueToConnectionLost
		//	}
		//}
		return ErrTimeout

	default:
		return err
	}
}

// applyEntryNormal apples an EntryNormal type raftpb request to the KVServer
func (ra *Replica) applyEntryNormal(ent *sm.Entry) {
	var ar *applyResult
	index := ra.consistIndex.ConsistentIndex()
	if ent.Index > index {
		// set the consistent index of current executing entry
		ra.consistIndex.SetConsistentApplyingIndex(ent.Index, ra.term)
		defer func() {
			// The txPostLockInsideApplyHook will not get called in some cases,
			// in which we should move the consistent index forward directly.
			newIndex := ra.consistIndex.ConsistentIndex()
			if newIndex < ent.Index {
				ra.consistIndex.SetConsistentIndex(ent.Index, ra.term)
			}
		}()
	}
	ra.lg.Debug("apply entry normal",
		zap.Uint64("consistent-index", index),
		zap.Uint64("entry-term", ra.term),
		zap.Uint64("entry-index", ent.Index))

	raftReq := pb.InternalRaftRequest{}
	if err := raftReq.Unmarshal(ent.Cmd); err != nil {
		ra.lg.Error("unmarshal raft entry",
			zap.Uint64("entry-term", ra.term),
			zap.Uint64("entry-index", ent.Index))
		return
	}
	ra.lg.Debug("applyEntryNormal", zap.Stringer("raftReq", &raftReq))

	id := raftReq.Header.ID
	needResult := ra.w.IsRegistered(id)
	if needResult || !noSideEffect(&raftReq) {
		ar = ra.apply.Apply(&raftReq)
	}

	if ar == nil {
		return
	}

	ra.w.Trigger(id, ar)

	//if !errors.Is(ar.err, errs.ErrNoSpace) || len(s.alarmStore.Get(pb.AlarmType_NOSPACE)) > 0 {
	//	s.w.Trigger(id, ar)
	//	return
	//}
	//
	//lg := sm.s.Logger()
	//lg.Warn(
	//	"message exceeded backend quota; raising alarm",
	//	zap.Int64("quota-size-bytes", sm.s.Cfg.QuotaBackendBytes),
	//	zap.String("quota-size", humanize.Bytes(uint64(sm.s.Cfg.QuotaBackendBytes))),
	//	zap.Error(ar.err),
	//)
	//
	//s.GoAttach(func() {
	//	//a := &pb.AlarmRequest{
	//	//	MemberID: uint64(s.ID()),
	//	//	Action:   pb.AlarmRequest_ACTIVATE,
	//	//	Alarm:    pb.AlarmType_NOSPACE,
	//	//}
	//	//s.raftRequest(s.ctx, pb.InternalRaftRequest{Alarm: a})
	//	s.w.Trigger(id, ar)
	//})
}

func (ra *Replica) Close() error {
	select {
	case <-ra.stopping:
		return nil
	default:
		close(ra.stopping)
	}

	<-ra.done

	ra.bemu.Lock()
	defer ra.bemu.Unlock()

	ra.be.ForceCommit()
	if err := ra.be.Close(); err != nil {
		ra.lg.Error("close backend", zap.Error(err))
	}

	return nil
}

func (ra *Replica) KV() mvcc.IWatchableKV { return ra.kv }
func (ra *Replica) Backend() backend.IBackend {
	ra.bemu.Lock()
	defer ra.bemu.Unlock()
	return ra.be
}

func (ra *Replica) AuthStore() auth.AuthStore { return ra.authStore }

func (ra *Replica) Logger() *zap.Logger {
	return ra.lg
}

func (ra *Replica) CleanUp() {
	ra.bemu.Lock()
	defer ra.bemu.Unlock()

	ra.be.ForceCommit()
	if err := ra.be.Close(); err != nil {
		ra.lg.Error("close backend", zap.Error(err))
	}
}

func (ra *Replica) ApplyWait() <-chan struct{} { return ra.applyWait.Wait(ra.getCommittedIndex()) }

// StoppingNotify returns a channel that receives an empty struct
// when the Replica is being stopped.
func (ra *Replica) StoppingNotify() <-chan struct{} { return ra.stopping }

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (ra *Replica) GoAttach(f func()) {
	ra.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer ra.wgMu.RUnlock()
	select {
	case <-ra.stopping:
		ra.lg.Warn("server has stopped; skipping GoAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	ra.wg.Add(1)
	go func() {
		defer ra.wg.Done()
		f()
	}()
}

func (ra *Replica) isLeader() bool {
	leaderID := ra.getLead()
	return leaderID != 0 && leaderID == ra.nodeID
}

func (ra *Replica) isReady() bool {
	return ra.getLead() > 0 && ra.getTerm() > 0
}

func (ra *Replica) ChangeNotify() {
	ra.changec <- struct{}{}
}

func (ra *Replica) setAppliedIndex(v uint64) {
	atomic.StoreUint64(&ra.appliedIndex, v)
}

func (ra *Replica) getAppliedIndex() uint64 {
	return atomic.LoadUint64(&ra.appliedIndex)
}

func (ra *Replica) setCommittedIndex(v uint64) {
	atomic.StoreUint64(&ra.appliedIndex, v)
}

func (ra *Replica) getCommittedIndex() uint64 {
	return atomic.LoadUint64(&ra.appliedIndex)
}

func (ra *Replica) setTerm(v uint64) {
	atomic.StoreUint64(&ra.term, v)
}

func (ra *Replica) getTerm() uint64 {
	return atomic.LoadUint64(&ra.term)
}

func (ra *Replica) setLead(v uint64) {
	atomic.StoreUint64(&ra.lead, v)
}

func (ra *Replica) getLead() uint64 {
	return atomic.LoadUint64(&ra.lead)
}

func (ra *Replica) ShardID() uint64 {
	return ra.id
}

func (ra *Replica) NodeID() uint64 {
	return ra.nodeID
}

func (ra *Replica) Leader() uint64 {
	return ra.getLead()
}

func (ra *Replica) AppliedIndex() uint64 {
	return ra.getAppliedIndex()
}

func (ra *Replica) CommittedIndex() uint64 {
	return ra.getCommittedIndex()
}

func (ra *Replica) Term() uint64 {
	return ra.getTerm()
}

func (ra *Replica) IsLearner() bool {
	return ra.isLearner
}

func (ra *Replica) IsMemberExist(id uint64) bool {
	ra.msmu.RLock()
	defer ra.msmu.RUnlock()
	_, ok := ra.members[id]
	return ok
}

func (ra *Replica) Cluster() *membership.RaftCluster {
	return ra.cluster
}
