package membership

import (
	"bytes"
	"fmt"
	"path"
	"strconv"

	"github.com/coreos/go-semver/semver"
	json "github.com/json-iterator/go"
	"github.com/olive-io/olive/server/mvcc/backend"
	"github.com/olive-io/olive/server/mvcc/buckets"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.uber.org/zap"
)

const (
	attributesSuffix     = "attributes"
	raftAttributesSuffix = "raftAttributes"

	// the prefix for storing membership related information in store provided by store pkg.
	storePrefix = "/0"
)

var (
	StoreMembersPrefix        = path.Join(storePrefix, "members")
	storeRemovedMembersPrefix = path.Join(storePrefix, "removed_members")
	errMemberAlreadyExist     = fmt.Errorf("member already exists")
	errMemberNotFound         = fmt.Errorf("member not found")
)

func unsafeSaveMemberToBackend(lg *zap.Logger, be backend.IBackend, m *Member) error {
	mkey := backendMemberKey(m.ID)
	mvalue, err := json.Marshal(m)
	if err != nil {
		lg.Panic("failed to marshal member", zap.Error(err))
	}

	tx := be.BatchTx()
	tx.LockInsideApply()
	defer tx.Unlock()
	if unsafeMemberExists(tx, mkey) {
		return errMemberAlreadyExist
	}
	tx.UnsafePut(buckets.Members, mkey, mvalue)
	return nil
}

// TrimClusterFromBackend removes all information about cluster (versions)
func TrimClusterFromBackend(be backend.IBackend) error {
	return nil
}

func unsafeDeleteMemberFromBackend(be backend.IBackend, id uint64) error {
	mkey := backendMemberKey(id)

	tx := be.BatchTx()
	tx.LockInsideApply()
	defer tx.Unlock()
	tx.UnsafePut(buckets.MembersRemoved, mkey, []byte("removed"))
	if !unsafeMemberExists(tx, mkey) {
		return errMemberNotFound
	}
	tx.UnsafeDelete(buckets.Members, mkey)
	return nil
}

func unsafeMemberExists(tx backend.IReadTx, mkey []byte) bool {
	var found bool
	tx.UnsafeForEach(buckets.Members, func(k, v []byte) error {
		if bytes.Equal(k, mkey) {
			found = true
		}
		return nil
	})
	return found
}

func readMembersFromBackend(lg *zap.Logger, be backend.IBackend) (map[uint64]*Member, map[uint64]bool, error) {
	members := make(map[uint64]*Member)
	removed := make(map[uint64]bool)

	tx := be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()
	err := tx.UnsafeForEach(buckets.Members, func(k, v []byte) error {
		memberId := mustParseMemberIDFromBytes(lg, k)
		m := &Member{ID: memberId}
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		members[memberId] = m
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read members from backend: %w", err)
	}

	err = tx.UnsafeForEach(buckets.MembersRemoved, func(k, v []byte) error {
		memberId := mustParseMemberIDFromBytes(lg, k)
		removed[memberId] = true
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read members_removed from backend: %w", err)
	}
	return members, removed, nil
}

func mustReadMembersFromBackend(lg *zap.Logger, be backend.IBackend) (map[uint64]*Member, map[uint64]bool) {
	members, removed, err := readMembersFromBackend(lg, be)
	if err != nil {
		lg.Panic("couldn't read members from backend", zap.Error(err))
	}
	return members, removed
}

// TrimMembershipFromBackend removes all information about members &
func TrimMembershipFromBackend(lg *zap.Logger, be backend.IBackend) error {
	lg.Info("Trimming membership information from the backend...")
	tx := be.BatchTx()
	tx.LockOutsideApply()
	defer tx.Unlock()
	err := tx.UnsafeForEach(buckets.Members, func(k, v []byte) error {
		tx.UnsafeDelete(buckets.Members, k)
		lg.Debug("Removed member from the backend",
			zap.Uint64("member", mustParseMemberIDFromBytes(lg, k)))
		return nil
	})
	if err != nil {
		return err
	}
	return tx.UnsafeForEach(buckets.MembersRemoved, func(k, v []byte) error {
		tx.UnsafeDelete(buckets.MembersRemoved, k)
		lg.Debug("Removed removed_member from the backend",
			zap.Uint64("member", mustParseMemberIDFromBytes(lg, k)))
		return nil
	})
}

func mustSaveClusterVersionToBackend(be backend.IBackend, ver *semver.Version) {
	ckey := backendClusterVersionKey()

	tx := be.BatchTx()
	tx.LockInsideApply()
	defer tx.Unlock()
	tx.UnsafePut(buckets.Cluster, ckey, []byte(ver.String()))
}

func mustSaveDowngradeToBackend(lg *zap.Logger, be backend.IBackend, downgrade *DowngradeInfo) {
	dkey := backendDowngradeKey()
	dvalue, err := json.Marshal(downgrade)
	if err != nil {
		lg.Panic("failed to marshal downgrade information", zap.Error(err))
	}
	tx := be.BatchTx()
	tx.LockInsideApply()
	defer tx.Unlock()
	tx.UnsafePut(buckets.Cluster, dkey, dvalue)
}

func backendMemberKey(id uint64) []byte {
	return []byte(fmt.Sprintf("%d", id))
}

func backendClusterVersionKey() []byte {
	return []byte("clusterVersion")
}

func backendDowngradeKey() []byte {
	return []byte("downgrade")
}

func mustCreateBackendBuckets(be backend.IBackend) {
	tx := be.BatchTx()
	tx.LockOutsideApply()
	defer tx.Unlock()
}

func MemberStoreKey(id types.ID) string {
	return path.Join(StoreMembersPrefix, id.String())
}

func StoreClusterVersionKey() string {
	return path.Join(storePrefix, "version")
}

func MemberAttributesStorePath(id types.ID) string {
	return path.Join(MemberStoreKey(id), attributesSuffix)
}

func mustParseMemberIDFromBytes(lg *zap.Logger, key []byte) uint64 {
	id, err := strconv.ParseUint(string(key), 10, 64)
	if err != nil {
		lg.Panic("failed to parse member id from key", zap.Error(err))
	}
	return id
}

func MustParseMemberIDFromKey(lg *zap.Logger, key string) uint64 {
	id, err := strconv.ParseUint(string(key), 10, 64)
	if err != nil {
		lg.Panic("failed to parse member id from key", zap.Error(err))
	}
	return id
}

func RemovedMemberStoreKey(id uint64) string {
	return path.Join(storeRemovedMembersPrefix, fmt.Sprintf("%d", id))
}
