package server

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/lni/dragonboat/v4"
	dbc "github.com/lni/dragonboat/v4/config"
	"github.com/olive-io/olive/server/config"
	"go.uber.org/zap"
)

func (s *OliveServer) StartReplica(cfg config.ShardConfig) error {

	rc := dbc.Config{
		ShardID:             cfg.ShardID,
		CheckQuorum:         true,
		PreVote:             s.PreVote,
		ElectionRTT:         s.ElectionTTL,
		HeartbeatRTT:        s.HeartBeatTTL,
		SnapshotEntries:     10,
		CompactionOverhead:  5,
		OrderedConfigChange: true,
		WaitReady:           true,
	}

	join := cfg.NewCluster

	members := map[uint64]string{}
	for key, urlText := range cfg.PeerURLs {
		URL, err := url.Parse(urlText.String())
		if err != nil {
			return fmt.Errorf("invalid url: %v", urlText.String())
		}
		replicaID := GenHash([]byte(key))
		peerAddress := URL.Host
		if peerAddress == s.ListenerPeerAddress {
			rc.ReplicaID = replicaID
		}
		members[replicaID] = peerAddress
	}

	start := time.Now()
	err := s.nh.StartOnDiskReplica(members, join, s.NewDiskKV, rc)
	if err != nil {
		return err
	}

	electionTimeout := cfg.ElectionTimeout
	if electionTimeout == 0 {
		electionTimeout = time.Duration(math.MaxInt64)
	}
	after := time.NewTimer(electionTimeout)
	defer after.Stop()
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return ErrStopped
		case <-after.C:
			return fmt.Errorf("wait shard ready: %w", ErrTimeout)
		case <-ticker.C:
		}

		leaderID, term, ok, e1 := s.nh.GetLeaderID(cfg.ShardID)
		if ok {
			s.lg.Info("start new shard",
				zap.Uint64("leader", leaderID),
				zap.Uint64("term", term),
				zap.Duration("duration", time.Now().Sub(start)))
			break
		}
		if e1 != nil {
			return fmt.Errorf("get leader %v", e1)
		}
	}

	return nil
}

func (s *OliveServer) processReplicaEvent() {
	for {
		select {
		case <-s.stopping:
			return
		case ch := <-s.raftEventCh:
			ra, ok := s.GetReplica(ch.ShardID)
			if ok {
				ra.setLead(ch.LeaderID)
				ra.setTerm(ch.Term)
				ra.ChangeNotify()
			}
		}
	}
}

func (s *OliveServer) ReplicaRequest(r *replicaRequest) {
	select {
	case <-s.stopping:
	case s.replicaRequest <- r:
	}
}

func (s *OliveServer) processReplicaRequest() {
	for {
		select {
		case <-s.stopping:
			return
		case req := <-s.replicaRequest:
			if rr := req.staleRead; rr != nil {
				s.requestStaleRead(req.shardID, rr)
			}
			if rr := req.syncRead; rr != nil {
				s.requestSyncRead(req.shardID, rr)
			}
			if rr := req.syncPropose; rr != nil {
				s.requestSyncPropose(req.shardID, rr)
			}
		}
	}
}

func (s *OliveServer) requestStaleRead(shardID uint64, r *replicaStaleRead) {
	result, err := s.nh.StaleRead(shardID, r.query)
	if err != nil {
		r.ec <- err
		return
	}
	r.rc <- result
}

func (s *OliveServer) requestSyncRead(shardID uint64, r *replicaSyncRead) {
	result, err := s.nh.SyncRead(r.ctx, shardID, r.query)
	if err != nil {
		r.ec <- err
		return
	}
	r.rc <- result
}

func (s *OliveServer) requestSyncPropose(shardID uint64, r *replicaSyncPropose) {
	session := s.nh.GetNoOPSession(shardID)
	result, err := s.nh.SyncPropose(r.ctx, session, r.data)
	if err != nil {
		r.ec <- err
		return
	}
	r.rc <- result
}

func (s *OliveServer) GetReplica(shardID uint64) (*Replica, bool) {
	s.rmu.RLock()
	defer s.rmu.RUnlock()
	ssm, ok := s.replicas[shardID]
	return ssm, ok
}

// TransferLeadership transfers the leader to the chosen transferee.
func (s *OliveServer) TransferLeadership() error {
	// TODO: TransferLeadership
	return nil
}

func (s *OliveServer) ShardView(shard uint64) {
	lg := s.Logger()
	opt := dragonboat.DefaultNodeHostInfoOption
	opt.SkipLogInfo = true
	nhi := s.nh.GetNodeHostInfo(opt)
	lg.Sugar().Infof("%+v", nhi)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	ms, e := s.nh.SyncGetShardMembership(ctx, shard)
	if e != nil {

	}
	lg.Sugar().Infof("%+v", ms)
}
