package membership

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-semver/semver"
	json "github.com/json-iterator/go"
	"github.com/olive-io/olive/api/raftpb"
	"github.com/olive-io/olive/pkg/netutil"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/server/mvcc/backend"
	"github.com/olive-io/olive/server/mvcc/buckets"
	"github.com/prometheus/client_golang/prometheus"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.uber.org/zap"
)

const maxLearners = 1

// RaftCluster is a list of Members that belong to the same raft cluster
type RaftCluster struct {
	lg *zap.Logger

	localID uint64
	cid     uint64

	be backend.IBackend

	sync.Mutex // guards the fields below
	version    *semver.Version
	members    map[uint64]*Member
	// removed contains the ids of removed members in the cluster.
	// removed id cannot be reused.
	removed map[uint64]bool

	downgradeInfo *DowngradeInfo
}

// ConfigChangeContext represents a context for confChange.
type ConfigChangeContext struct {
	Member
	// IsPromote indicates if the config change is for promoting a learner member.
	// This flag is needed because both adding a new member and promoting a learner member
	// uses the same config change type 'ConfChangeAddNode'.
	IsPromote bool `json:"isPromote"`
}

// NewClusterFromURLsMap creates a new raft cluster using provided urls map. Currently, it does not support creating
// cluster with raft learner member.
func NewClusterFromURLsMap(lg *zap.Logger, token string, urlsmap types.URLsMap) (*RaftCluster, error) {
	c := NewCluster(lg)
	for name, urls := range urlsmap {
		m := NewMember(name, urls, token, nil)
		if _, ok := c.members[m.ID]; ok {
			return nil, fmt.Errorf("member exists with identical ID %v", m)
		}
		if uint64(m.ID) == 0 {
			return nil, fmt.Errorf("cannot use %x as member id", 0)
		}
		c.members[m.ID] = m
	}
	c.genID()
	return c, nil
}

func NewClusterFromMembers(lg *zap.Logger, id uint64, membs []*Member) *RaftCluster {
	c := NewCluster(lg)
	c.cid = id
	for _, m := range membs {
		c.members[m.ID] = m
	}
	return c
}

func NewCluster(lg *zap.Logger) *RaftCluster {
	if lg == nil {
		lg = zap.NewNop()
	}
	return &RaftCluster{
		lg:            lg,
		members:       make(map[uint64]*Member),
		removed:       make(map[uint64]bool),
		downgradeInfo: &DowngradeInfo{Enabled: false},
	}
}

func (c *RaftCluster) ID() uint64 { return c.cid }

func (c *RaftCluster) Members() []*Member {
	c.Lock()
	defer c.Unlock()
	var ms MembersByID
	for _, m := range c.members {
		ms = append(ms, m.Clone())
	}
	sort.Sort(ms)
	return []*Member(ms)
}

func (c *RaftCluster) Member(id uint64) *Member {
	c.Lock()
	defer c.Unlock()
	return c.members[id].Clone()
}

func (c *RaftCluster) VotingMembers() []*Member {
	c.Lock()
	defer c.Unlock()
	var ms MembersByID
	for _, m := range c.members {
		if !m.IsLearner {
			ms = append(ms, m.Clone())
		}
	}
	sort.Sort(ms)
	return []*Member(ms)
}

// MemberByName returns a Member with the given name if exists.
// If more than one member has the given name, it will panic.
func (c *RaftCluster) MemberByName(name string) *Member {
	c.Lock()
	defer c.Unlock()
	var memb *Member
	for _, m := range c.members {
		if m.Name == name {
			if memb != nil {
				c.lg.Panic("two member with same name found", zap.String("name", name))
			}
			memb = m
		}
	}
	return memb.Clone()
}

func (c *RaftCluster) MemberIDs() []uint64 {
	c.Lock()
	defer c.Unlock()
	var ids []uint64
	for _, m := range c.members {
		ids = append(ids, m.ID)
	}
	//sort.Sort(ids)
	return ids
}

func (c *RaftCluster) IsIDRemoved(id uint64) bool {
	c.Lock()
	defer c.Unlock()
	return c.removed[id]
}

// PeerURLs returns a list of all peer addresses.
// The returned list is sorted in ascending lexicographical order.
func (c *RaftCluster) PeerURLs() []string {
	c.Lock()
	defer c.Unlock()
	urls := make([]string, 0)
	for _, p := range c.members {
		urls = append(urls, p.PeerURLs...)
	}
	sort.Strings(urls)
	return urls
}

// ClientURLs returns a list of all client addresses.
// The returned list is sorted in ascending lexicographical order.
func (c *RaftCluster) ClientURLs() []string {
	c.Lock()
	defer c.Unlock()
	urls := make([]string, 0)
	for _, p := range c.members {
		urls = append(urls, p.ClientURLs...)
	}
	sort.Strings(urls)
	return urls
}

func (c *RaftCluster) String() string {
	c.Lock()
	defer c.Unlock()
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "{ClusterID:%d ", c.cid)
	var ms []string
	for _, m := range c.members {
		ms = append(ms, fmt.Sprintf("%+v", m))
	}
	fmt.Fprintf(b, "Members:[%s] ", strings.Join(ms, " "))
	var ids []string
	for id := range c.removed {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	fmt.Fprintf(b, "RemovedMemberIDs:[%s]}", strings.Join(ids, " "))
	return b.String()
}

func (c *RaftCluster) genID() {
	mIDs := c.MemberIDs()
	b := make([]byte, 8*len(mIDs))
	for i, id := range mIDs {
		binary.BigEndian.PutUint64(b[8*i:], uint64(id))
	}
	hash := sha1.Sum(b)
	c.cid = binary.BigEndian.Uint64(hash[:8])
}

func (c *RaftCluster) SetID(localID, cid uint64) {
	c.localID = localID
	c.cid = cid
}

func (c *RaftCluster) SetBackend(be backend.IBackend) {
	c.be = be
	mustCreateBackendBuckets(c.be)
}

func (c *RaftCluster) Recover(onSet func(*zap.Logger, *semver.Version)) {
	c.Lock()
	defer c.Unlock()

	c.members, c.removed = membersFromBackend(c.lg, c.be)

	if c.be != nil {
		c.downgradeInfo = downgradeInfoFromBackend(c.lg, c.be)
	}
	d := &DowngradeInfo{Enabled: false}
	if c.downgradeInfo != nil {
		d = &DowngradeInfo{Enabled: c.downgradeInfo.Enabled, TargetVersion: c.downgradeInfo.TargetVersion}
	}
	mustDetectDowngrade(c.lg, c.version, d)
	onSet(c.lg, c.version)

	for _, m := range c.members {
		c.lg.Info(
			"recovered/added member from store",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.Uint64("recovered-remote-peer-id", m.ID),
			zap.Strings("recovered-remote-peer-urls", m.PeerURLs),
		)
	}
	if c.version != nil {
		c.lg.Info(
			"set cluster version from store",
			zap.String("cluster-version", version.Cluster(c.version.String())),
		)
	}
}

// ValidateConfigurationChange takes a proposed ConfChange and
// ensures that it is still valid.
func (c *RaftCluster) ValidateConfigurationChange(cc raftpb.ConfChange) error {
	members, removed := membersFromBackend(c.lg, c.be)
	id := cc.NodeID
	if removed[id] {
		return ErrIDRemoved
	}
	switch cc.Type {
	case raftpb.ConfChangeType_ConfChangeAddNode, raftpb.ConfChangeType_ConfChangeAddLearnerNode:
		confChangeContext := new(ConfigChangeContext)
		if err := json.Unmarshal(cc.Context, confChangeContext); err != nil {
			c.lg.Panic("failed to unmarshal confChangeContext", zap.Error(err))
		}

		if confChangeContext.IsPromote { // promoting a learner member to voting member
			if members[id] == nil {
				return ErrIDNotFound
			}
			if !members[id].IsLearner {
				return ErrMemberNotLearner
			}
		} else { // adding a new member
			if members[id] != nil {
				return ErrIDExists
			}

			urls := make(map[string]bool)
			for _, m := range members {
				for _, u := range m.PeerURLs {
					urls[u] = true
				}
			}
			for _, u := range confChangeContext.Member.PeerURLs {
				if urls[u] {
					return ErrPeerURLexists
				}
			}

			if confChangeContext.Member.IsLearner { // the new member is a learner
				numLearners := 0
				for _, m := range members {
					if m.IsLearner {
						numLearners++
					}
				}
				if numLearners+1 > maxLearners {
					return ErrTooManyLearners
				}
			}
		}
	case raftpb.ConfChangeType_ConfChangeRemoveNode:
		if members[id] == nil {
			return ErrIDNotFound
		}

	case raftpb.ConfChangeType_ConfChangeUpdateNode:
		if members[id] == nil {
			return ErrIDNotFound
		}
		urls := make(map[string]bool)
		for _, m := range members {
			if m.ID == id {
				continue
			}
			for _, u := range m.PeerURLs {
				urls[u] = true
			}
		}
		m := new(Member)
		if err := json.Unmarshal(cc.Context, m); err != nil {
			c.lg.Panic("failed to unmarshal member", zap.Error(err))
		}
		for _, u := range m.PeerURLs {
			if urls[u] {
				return ErrPeerURLexists
			}
		}

	default:
		c.lg.Panic("unknown ConfChange type", zap.String("type", cc.Type.String()))
	}
	return nil
}

// AddMember adds a new Member into the cluster, and saves the given member's
// raftAttributes into the store. The given member should have empty attributes.
// A Member with a matching id must not exist.
func (c *RaftCluster) AddMember(m *Member) {
	c.Lock()
	defer c.Unlock()

	var err error
	if c.be != nil {
		err = unsafeSaveMemberToBackend(c.lg, c.be, m)
		if err != nil && !errors.Is(err, errMemberAlreadyExist) {
			c.lg.Panic(
				"failed to save member to backend",
				zap.Uint64("member-id", m.ID),
				zap.Error(err),
			)
		}
	}
	// Panic if both storeV2 and backend report member already exist.
	if err != nil || c.be == nil {
		c.lg.Panic(
			"failed to save member to store",
			zap.Uint64("member-id", m.ID),
			zap.Error(err),
		)
	}

	c.members[m.ID] = m

	c.lg.Info(
		"added member",
		zap.Uint64("cluster-id", c.cid),
		zap.Uint64("local-member-id", c.localID),
		zap.Uint64("added-peer-id", m.ID),
		zap.Strings("added-peer-peer-urls", m.PeerURLs),
	)
}

// RemoveMember removes a member from the store.
// The given id MUST exist, or the function panics.
func (c *RaftCluster) RemoveMember(id uint64) {
	c.Lock()
	defer c.Unlock()
	var err error
	if c.be != nil {
		err = unsafeDeleteMemberFromBackend(c.be, id)
		if err != nil && !errors.Is(err, errMemberNotFound) {
			c.lg.Panic(
				"failed to delete member from backend",
				zap.Uint64("member-id", id),
				zap.Error(err),
			)
		}
	}
	// Panic if both storeV2 and backend report member not found.
	if err != nil || c.be == nil {
		c.lg.Panic(
			"failed to delete member from store",
			zap.Uint64("member-id", id),
			zap.Error(err),
		)
	}

	m, ok := c.members[id]
	delete(c.members, id)
	c.removed[id] = true

	if ok {
		c.lg.Info(
			"removed member",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.Uint64("removed-remote-peer-id", id),
			zap.Strings("removed-remote-peer-urls", m.PeerURLs),
		)
	} else {
		c.lg.Warn(
			"skipped removing already removed member",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.Uint64("removed-remote-peer-id", id),
		)
	}
}

func (c *RaftCluster) UpdateAttributes(id uint64, attr Attributes) {
	c.Lock()
	defer c.Unlock()

	if m, ok := c.members[id]; ok {
		m.Attributes = attr
		if c.be != nil {
			unsafeSaveMemberToBackend(c.lg, c.be, m)
		}
		return
	}

	_, ok := c.removed[id]
	if !ok {
		c.lg.Panic(
			"failed to update; member unknown",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.Uint64("unknown-remote-peer-id", id),
		)
	}

	c.lg.Warn(
		"skipped attributes update of removed member",
		zap.Uint64("cluster-id", c.cid),
		zap.Uint64("local-member-id", c.localID),
		zap.Uint64("updated-peer-id", id),
	)
}

// PromoteMember marks the member's IsLearner RaftAttributes to false.
func (c *RaftCluster) PromoteMember(id uint64) {
	c.Lock()
	defer c.Unlock()

	c.members[id].RaftAttributes.IsLearner = false
	if c.be != nil {
		unsafeSaveMemberToBackend(c.lg, c.be, c.members[id])
	}

	c.lg.Info(
		"promote member",
		zap.Uint64("cluster-id", c.cid),
		zap.Uint64("local-member-id", c.localID),
	)
}

func (c *RaftCluster) UpdateRaftAttributes(id uint64, raftAttr RaftAttributes) {
	c.Lock()
	defer c.Unlock()

	c.members[id].RaftAttributes = raftAttr
	if c.be != nil {
		unsafeSaveMemberToBackend(c.lg, c.be, c.members[id])
	}

	c.lg.Info(
		"updated member",
		zap.Uint64("cluster-id", c.cid),
		zap.Uint64("local-member-id", c.localID),
		zap.Uint64("updated-remote-peer-id", id),
		zap.Strings("updated-remote-peer-urls", raftAttr.PeerURLs),
	)
}

func (c *RaftCluster) Version() *semver.Version {
	c.Lock()
	defer c.Unlock()
	if c.version == nil {
		return nil
	}
	return semver.Must(semver.NewVersion(c.version.String()))
}

func (c *RaftCluster) SetVersion(ver *semver.Version, onSet func(*zap.Logger, *semver.Version)) {
	c.Lock()
	defer c.Unlock()
	if c.version != nil {
		c.lg.Info(
			"updated cluster version",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.String("from", version.Cluster(c.version.String())),
			zap.String("to", version.Cluster(ver.String())),
		)
	} else {
		c.lg.Info(
			"set initial cluster version",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
			zap.String("cluster-version", version.Cluster(ver.String())),
		)
	}
	oldVer := c.version
	c.version = ver
	mustDetectDowngrade(c.lg, c.version, c.downgradeInfo)
	if c.be != nil {
		mustSaveClusterVersionToBackend(c.be, ver)
	}
	if oldVer != nil {
		ClusterVersionMetrics.With(prometheus.Labels{"cluster_version": version.Cluster(oldVer.String())}).Set(0)
	}
	ClusterVersionMetrics.With(prometheus.Labels{"cluster_version": version.Cluster(ver.String())}).Set(1)
	onSet(c.lg, ver)
}

func (c *RaftCluster) IsReadyToAddVotingMember() bool {
	nmembers := 1
	nstarted := 0

	for _, member := range c.VotingMembers() {
		if member.IsStarted() {
			nstarted++
		}
		nmembers++
	}

	if nstarted == 1 && nmembers == 2 {
		// a case of adding a new node to 1-member cluster for restoring cluster data
		c.lg.Debug("number of started member is 1; can accept add member request")
		return true
	}

	nquorum := nmembers/2 + 1
	if nstarted < nquorum {
		c.lg.Warn(
			"rejecting member add; started member will be less than quorum",
			zap.Int("number-of-started-member", nstarted),
			zap.Int("quorum", nquorum),
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
		)
		return false
	}

	return true
}

func (c *RaftCluster) IsReadyToRemoveVotingMember(id uint64) bool {
	nmembers := 0
	nstarted := 0

	for _, member := range c.VotingMembers() {
		if uint64(member.ID) == id {
			continue
		}

		if member.IsStarted() {
			nstarted++
		}
		nmembers++
	}

	nquorum := nmembers/2 + 1
	if nstarted < nquorum {
		c.lg.Warn(
			"rejecting member remove; started member will be less than quorum",
			zap.Int("number-of-started-member", nstarted),
			zap.Int("quorum", nquorum),
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
		)
		return false
	}

	return true
}

func (c *RaftCluster) IsReadyToPromoteMember(id uint64) bool {
	nmembers := 1 // We count the learner to be promoted for the future quorum
	nstarted := 1 // and we also count it as started.

	for _, member := range c.VotingMembers() {
		if member.IsStarted() {
			nstarted++
		}
		nmembers++
	}

	nquorum := nmembers/2 + 1
	if nstarted < nquorum {
		c.lg.Warn(
			"rejecting member promote; started member will be less than quorum",
			zap.Int("number-of-started-member", nstarted),
			zap.Int("quorum", nquorum),
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
		)
		return false
	}

	return true
}

func membersFromBackend(lg *zap.Logger, be backend.IBackend) (map[uint64]*Member, map[uint64]bool) {
	return mustReadMembersFromBackend(lg, be)
}

// The field is populated since etcd v3.5.
func downgradeInfoFromBackend(lg *zap.Logger, be backend.IBackend) *DowngradeInfo {
	dkey := backendDowngradeKey()
	tx := be.ReadTx()
	tx.Lock()
	defer tx.Unlock()
	keys, vals, _ := tx.UnsafeRange(buckets.Cluster, dkey, nil, 0)
	if len(keys) == 0 {
		return nil
	}

	if len(keys) != 1 {
		lg.Panic(
			"unexpected number of keys when getting cluster version from backend",
			zap.Int("number-of-key", len(keys)),
		)
	}
	var d DowngradeInfo
	if err := json.Unmarshal(vals[0], &d); err != nil {
		lg.Panic("failed to unmarshal downgrade information", zap.Error(err))
	}

	// verify the downgrade info from backend
	if d.Enabled {
		if _, err := semver.NewVersion(d.TargetVersion); err != nil {
			lg.Panic(
				"unexpected version format of the downgrade target version from backend",
				zap.String("target-version", d.TargetVersion),
			)
		}
	}
	return &d
}

// ValidateClusterAndAssignIDs validates the local cluster by matching the PeerURLs
// with the existing cluster. If the validation succeeds, it assigns the IDs
// from the existing cluster to the local cluster.
// If the validation fails, an error will be returned.
func ValidateClusterAndAssignIDs(lg *zap.Logger, local *RaftCluster, existing *RaftCluster) error {
	ems := existing.Members()
	lms := local.Members()
	if len(ems) != len(lms) {
		return fmt.Errorf("member count is unequal")
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	for i := range ems {
		var err error
		ok := false
		for j := range lms {
			if ok, err = netutil.URLStringsEqual(ctx, lg, ems[i].PeerURLs, lms[j].PeerURLs); ok {
				lms[j].ID = ems[i].ID
				break
			}
		}
		if !ok {
			return fmt.Errorf("PeerURLs: no match found for existing member (%v, %v), last resolver error (%v)", ems[i].ID, ems[i].PeerURLs, err)
		}
	}
	local.members = make(map[uint64]*Member)
	for _, m := range lms {
		local.members[m.ID] = m
	}
	return nil
}

func IsValidVersionChange(cv *semver.Version, lv *semver.Version) bool {
	cv = &semver.Version{Major: cv.Major, Minor: cv.Minor}
	lv = &semver.Version{Major: lv.Major, Minor: lv.Minor}

	if isValidDowngrade(cv, lv) || (cv.Major == lv.Major && cv.LessThan(*lv)) {
		return true
	}
	return false
}

// IsLocalMemberLearner returns if the local member is raft learner
func (c *RaftCluster) IsLocalMemberLearner() bool {
	c.Lock()
	defer c.Unlock()
	localMember, ok := c.members[c.localID]
	if !ok {
		c.lg.Panic(
			"failed to find local ID in cluster members",
			zap.Uint64("cluster-id", c.cid),
			zap.Uint64("local-member-id", c.localID),
		)
	}
	return localMember.IsLearner
}

// DowngradeInfo returns the downgrade status of the cluster
func (c *RaftCluster) DowngradeInfo() *DowngradeInfo {
	c.Lock()
	defer c.Unlock()
	if c.downgradeInfo == nil {
		return &DowngradeInfo{Enabled: false}
	}
	d := &DowngradeInfo{Enabled: c.downgradeInfo.Enabled, TargetVersion: c.downgradeInfo.TargetVersion}
	return d
}

func (c *RaftCluster) SetDowngradeInfo(d *DowngradeInfo) {
	c.Lock()
	defer c.Unlock()

	mustSaveDowngradeToBackend(c.lg, c.be, d)
	c.downgradeInfo = d

	if d.Enabled {
		c.lg.Info(
			"The server is ready to downgrade",
			zap.String("target-version", d.TargetVersion),
			zap.String("server-version", version.Version),
		)
	}
}

// IsMemberExist returns if the member with the given id exists in cluster.
func (c *RaftCluster) IsMemberExist(id uint64) bool {
	c.Lock()
	defer c.Unlock()
	_, ok := c.members[id]
	return ok
}

// VotingMemberIDs returns the ID of voting members in cluster.
func (c *RaftCluster) VotingMemberIDs() []uint64 {
	c.Lock()
	defer c.Unlock()
	var ids []uint64
	for _, m := range c.members {
		if !m.IsLearner {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// PushMembershipToStorage is overriding storage information about cluster's
// members, such that they fully reflect internal RaftCluster's storage.
func (c *RaftCluster) PushMembershipToStorage() {
	if c.be != nil {
		TrimMembershipFromBackend(c.lg, c.be)
		for _, m := range c.members {
			unsafeSaveMemberToBackend(c.lg, c.be, m)
		}
	}
}
