package runner

import (
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	"github.com/olive-io/olive/runner/backend"
	"go.uber.org/zap"
)

type MultiRaftGroup struct {
	nh *dragonboat.NodeHost

	leaderCh chan raftio.LeaderInfo

	be backend.IBackend

	lg *zap.Logger

	stopping <-chan struct{}
	done     chan struct{}
}

func (r *Runner) newMultiRaftGroup() (*MultiRaftGroup, error) {
	cfg := r.Config
	lg := cfg.Logger
	if lg == nil {
		lg = zap.NewExample()
	}

	be := newRegionBackend(&cfg)

	leaderCh := make(chan raftio.LeaderInfo, 10)
	el := newEventListener(leaderCh)

	sl := newSystemListener()

	dir := cfg.RegionDir()
	nhConfig := config.NodeHostConfig{
		NodeHostDir:         dir,
		RTTMillisecond:      cfg.RaftRTTMillisecond,
		RaftAddress:         cfg.PeerListen,
		EnableMetrics:       true,
		RaftEventListener:   el,
		SystemEventListener: sl,
		NotifyCommit:        true,
	}

	lg.Debug("start multi raft group",
		zap.String("module", "dragonboat"),
		zap.String("dir", dir),
		zap.String("listen", cfg.PeerListen))

	nh, err := dragonboat.NewNodeHost(nhConfig)
	if err != nil {
		return nil, err
	}

	mrg := &MultiRaftGroup{
		nh:       nh,
		leaderCh: leaderCh,
		be:       be,
		lg:       lg,
		stopping: r.StoppingNotify(),
		done:     make(chan struct{}, 1),
	}

	go mrg.run()
	return mrg, nil
}

func (mrg *MultiRaftGroup) run() {
	defer mrg.Stop()
	for {
		select {
		case <-mrg.stopping:
			return
		case leaderInfo := <-mrg.leaderCh:
			_ = leaderInfo
		}
	}
}

func (mrg *MultiRaftGroup) Stop() {
	mrg.nh.Close()

	select {
	case <-mrg.done:
	default:
		close(mrg.done)
	}
}

func (mrg *MultiRaftGroup) CreateRegion() error {
	//mrg.nh.StartOnDiskReplica()

	return nil
}
