/*
Copyright 2024 The olive Authors

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

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/metrics"
)

var (
	cleanAssumedPeriod = 1 * time.Second
)

// New returns a Cache implementation.
// It automatically starts a go routine that manages expiration of assumed regions.
// "ttl" is how long the assumed region will get expired.
// "ctx" is the context that would close the background goroutine.
func New(ctx context.Context, ttl time.Duration) Cache {
	logger := klog.FromContext(ctx)
	cache := newCache(ctx, ttl, cleanAssumedPeriod)
	cache.run(logger)
	return cache
}

// runnerInfoListItem holds a RunnerInfo pointer and acts as an item in a doubly
// linked list. When a RunnerInfo is updated, it goes to the head of the list.
// The items closer to the head are the most recently updated items.
type runnerInfoListItem struct {
	info *framework.RunnerInfo
	next *runnerInfoListItem
	prev *runnerInfoListItem
}

type cacheImpl struct {
	stop   <-chan struct{}
	ttl    time.Duration
	period time.Duration

	// This mutex guards all fields within this cache struct.
	mu sync.RWMutex
	// a set of assumed region keys.
	// The key could further be used to get an entry in regionStates.
	assumedRegions sets.Set[string]
	// a map from region key to regionState.
	regionStates map[string]*regionState
	runners      map[string]*runnerInfoListItem
	// headRunner points to the most recently updated RunnerInfo in "runners". It is the
	// head of the linked list.
	headRunner *runnerInfoListItem
	runnerTree *runnerTree
	// A map from image name to its ImageStateSummary.
	imageStates map[string]*framework.ImageStateSummary
}

type regionState struct {
	region *corev1.Region
	// Used by assumedRegion to determinate expiration.
	// If deadline is nil, assumedRegion will never expire.
	deadline *time.Time
	// Used to block cache from expiring assumedRegion if binding still runs
	bindingFinished bool
}

func newCache(ctx context.Context, ttl, period time.Duration) *cacheImpl {
	logger := klog.FromContext(ctx)
	return &cacheImpl{
		ttl:    ttl,
		period: period,
		stop:   ctx.Done(),

		runners:        make(map[string]*runnerInfoListItem),
		runnerTree:     newRunnerTree(logger, nil),
		assumedRegions: sets.New[string](),
		regionStates:   make(map[string]*regionState),
		imageStates:    make(map[string]*framework.ImageStateSummary),
	}
}

// newRunnerInfoListItem initializes a new runnerInfoListItem.
func newRunnerInfoListItem(ni *framework.RunnerInfo) *runnerInfoListItem {
	return &runnerInfoListItem{
		info: ni,
	}
}

// moveRunnerInfoToHead moves a RunnerInfo to the head of "cache.runners" doubly
// linked list. The head is the most recently updated RunnerInfo.
// We assume cache lock is already acquired.
func (cache *cacheImpl) moveRunnerInfoToHead(logger klog.Logger, name string) {
	ni, ok := cache.runners[name]
	if !ok {
		logger.Error(nil, "No runner info with given name found in the cache", "runner", klog.KRef("", name))
		return
	}
	// if the runner info list item is already at the head, we are done.
	if ni == cache.headRunner {
		return
	}

	if ni.prev != nil {
		ni.prev.next = ni.next
	}
	if ni.next != nil {
		ni.next.prev = ni.prev
	}
	if cache.headRunner != nil {
		cache.headRunner.prev = ni
	}
	ni.next = cache.headRunner
	ni.prev = nil
	cache.headRunner = ni
}

// removeRunnerInfoFromList removes a RunnerInfo from the "cache.runners" doubly
// linked list.
// We assume cache lock is already acquired.
func (cache *cacheImpl) removeRunnerInfoFromList(logger klog.Logger, name string) {
	ni, ok := cache.runners[name]
	if !ok {
		logger.Error(nil, "No runner info with given name found in the cache", "runner", klog.KRef("", name))
		return
	}

	if ni.prev != nil {
		ni.prev.next = ni.next
	}
	if ni.next != nil {
		ni.next.prev = ni.prev
	}
	// if the removed item was at the head, we must update the head.
	if ni == cache.headRunner {
		cache.headRunner = ni.next
	}
	delete(cache.runners, name)
}

// Dump produces a dump of the current scheduler cache. This is used for
// debugging purposes only and shouldn't be confused with UpdateSnapshot
// function.
// This method is expensive, and should be only used in non-critical path.
func (cache *cacheImpl) Dump() *Dump {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	runners := make(map[string]*framework.RunnerInfo, len(cache.runners))
	for k, v := range cache.runners {
		runners[k] = v.info.Snapshot()
	}

	return &Dump{
		Runners:        runners,
		AssumedRegions: cache.assumedRegions.Union(nil),
	}
}

// UpdateSnapshot takes a snapshot of cached RunnerInfo map. This is called at
// beginning of every scheduling cycle.
// The snapshot only includes Runners that are not deleted at the time this function is called.
// runnerInfo.Runner() is guaranteed to be not nil for all the runners in the snapshot.
// This function tracks generation number of RunnerInfo and updates only the
// entries of an existing snapshot that have changed after the snapshot was taken.
func (cache *cacheImpl) UpdateSnapshot(logger klog.Logger, runnerSnapshot *Snapshot) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Get the last generation of the snapshot.
	snapshotGeneration := runnerSnapshot.generation

	// RunnerInfoList and HaveRegionsWithAffinityRunnerInfoList must be re-created if a runner was added
	// or removed from the cache.
	updateAllLists := false
	// HaveRegionsWithAffinityRunnerInfoList must be re-created if a runner changed its
	// status from having regions with affinity to NOT having regions with affinity or the other
	// way around.
	updateRunnersHaveRegionsWithAffinity := false
	// HaveRegionsWithRequiredAntiAffinityRunnerInfoList must be re-created if a runner changed its
	// status from having regions with required anti-affinity to NOT having regions with required
	// anti-affinity or the other way around.
	updateRunnersHaveRegionsWithRequiredAntiAffinity := false
	// usedPVCSet must be re-created whenever the head runner generation is greater than
	// last snapshot generation.
	updateUsedPVCSet := false

	// Start from the head of the RunnerInfo doubly linked list and update snapshot
	// of RunnerInfos updated after the last snapshot.
	for runner := cache.headRunner; runner != nil; runner = runner.next {
		if runner.info.Generation <= snapshotGeneration {
			// all the runners are updated before the existing snapshot. We are done.
			break
		}
		if np := runner.info.Runner(); np != nil {
			existing, ok := runnerSnapshot.runnerInfoMap[np.Name]
			if !ok {
				updateAllLists = true
				existing = &framework.RunnerInfo{}
				runnerSnapshot.runnerInfoMap[np.Name] = existing
			}
			clone := runner.info.Snapshot()
			// We track runners that have regions with affinity, here we check if this runner changed its
			// status from having regions with affinity to NOT having regions with affinity or the other
			// way around.
			if (len(existing.RegionsWithAffinity) > 0) != (len(clone.RegionsWithAffinity) > 0) {
				updateRunnersHaveRegionsWithAffinity = true
			}
			if (len(existing.RegionsWithRequiredAntiAffinity) > 0) != (len(clone.RegionsWithRequiredAntiAffinity) > 0) {
				updateRunnersHaveRegionsWithRequiredAntiAffinity = true
			}
			if !updateUsedPVCSet {
				if len(existing.PVCRefCounts) != len(clone.PVCRefCounts) {
					updateUsedPVCSet = true
				} else {
					for pvcKey := range clone.PVCRefCounts {
						if _, found := existing.PVCRefCounts[pvcKey]; !found {
							updateUsedPVCSet = true
							break
						}
					}
				}
			}
			// We need to preserve the original pointer of the RunnerInfo struct since it
			// is used in the RunnerInfoList, which we may not update.
			*existing = *clone
		}
	}
	// Update the snapshot generation with the latest RunnerInfo generation.
	if cache.headRunner != nil {
		runnerSnapshot.generation = cache.headRunner.info.Generation
	}

	// Comparing to regions in runnerTree.
	// Deleted runners get removed from the tree, but they might remain in the runners map
	// if they still have non-deleted Regions.
	if len(runnerSnapshot.runnerInfoMap) > cache.runnerTree.numRunners {
		cache.removeDeletedRunnersFromSnapshot(runnerSnapshot)
		updateAllLists = true
	}

	if updateAllLists || updateRunnersHaveRegionsWithAffinity || updateRunnersHaveRegionsWithRequiredAntiAffinity || updateUsedPVCSet {
		cache.updateRunnerInfoSnapshotList(logger, runnerSnapshot, updateAllLists)
	}

	if len(runnerSnapshot.runnerInfoList) != cache.runnerTree.numRunners {
		errMsg := fmt.Sprintf("snapshot state is not consistent, length of RunnerInfoList=%v not equal to length of runners in tree=%v "+
			", length of RunnerInfoMap=%v, length of runners in cache=%v"+
			", trying to recover",
			len(runnerSnapshot.runnerInfoList), cache.runnerTree.numRunners,
			len(runnerSnapshot.runnerInfoMap), len(cache.runners))
		logger.Error(nil, errMsg)
		// We will try to recover by re-creating the lists for the next scheduling cycle, but still return an
		// error to surface the problem, the error will likely cause a failure to the current scheduling cycle.
		cache.updateRunnerInfoSnapshotList(logger, runnerSnapshot, true)
		return fmt.Errorf(errMsg)
	}

	return nil
}

func (cache *cacheImpl) updateRunnerInfoSnapshotList(logger klog.Logger, snapshot *Snapshot, updateAll bool) {
	snapshot.haveRegionsWithAffinityRunnerInfoList = make([]*framework.RunnerInfo, 0, cache.runnerTree.numRunners)
	snapshot.haveRegionsWithRequiredAntiAffinityRunnerInfoList = make([]*framework.RunnerInfo, 0, cache.runnerTree.numRunners)
	if updateAll {
		// Take a snapshot of the runners order in the tree
		snapshot.runnerInfoList = make([]*framework.RunnerInfo, 0, cache.runnerTree.numRunners)
		runnersList, err := cache.runnerTree.list()
		if err != nil {
			logger.Error(err, "Error occurred while retrieving the list of names of the runners from runner tree")
		}
		for _, runnerName := range runnersList {
			if runnerInfo := snapshot.runnerInfoMap[runnerName]; runnerInfo != nil {
				snapshot.runnerInfoList = append(snapshot.runnerInfoList, runnerInfo)
				if len(runnerInfo.RegionsWithAffinity) > 0 {
					snapshot.haveRegionsWithAffinityRunnerInfoList = append(snapshot.haveRegionsWithAffinityRunnerInfoList, runnerInfo)
				}
				if len(runnerInfo.RegionsWithRequiredAntiAffinity) > 0 {
					snapshot.haveRegionsWithRequiredAntiAffinityRunnerInfoList = append(snapshot.haveRegionsWithRequiredAntiAffinityRunnerInfoList, runnerInfo)
				}
				for key := range runnerInfo.PVCRefCounts {
				}
			} else {
				logger.Error(nil, "Runner exists in runnerTree but not in RunnerInfoMap, this should not happen", "runner", klog.KRef("", runnerName))
			}
		}
	} else {
		for _, runnerInfo := range snapshot.runnerInfoList {
			if len(runnerInfo.RegionsWithAffinity) > 0 {
				snapshot.haveRegionsWithAffinityRunnerInfoList = append(snapshot.haveRegionsWithAffinityRunnerInfoList, runnerInfo)
			}
			if len(runnerInfo.RegionsWithRequiredAntiAffinity) > 0 {
				snapshot.haveRegionsWithRequiredAntiAffinityRunnerInfoList = append(snapshot.haveRegionsWithRequiredAntiAffinityRunnerInfoList, runnerInfo)
			}
			for key := range runnerInfo.PVCRefCounts {
				snapshot.usedPVCSet.Insert(key)
			}
		}
	}
}

// If certain runners were deleted after the last snapshot was taken, we should remove them from the snapshot.
func (cache *cacheImpl) removeDeletedRunnersFromSnapshot(snapshot *Snapshot) {
	toDelete := len(snapshot.runnerInfoMap) - cache.runnerTree.numRunners
	for name := range snapshot.runnerInfoMap {
		if toDelete <= 0 {
			break
		}
		if n, ok := cache.runners[name]; !ok || n.info.Runner() == nil {
			delete(snapshot.runnerInfoMap, name)
			toDelete--
		}
	}
}

// RunnerCount returns the number of runners in the cache.
// DO NOT use outside of tests.
func (cache *cacheImpl) RunnerCount() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return len(cache.runners)
}

// RegionCount returns the number of regions in the cache (including those from deleted runners).
// DO NOT use outside of tests.
func (cache *cacheImpl) RegionCount() (int, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	// regionFilter is expected to return true for most or all of the regions. We
	// can avoid expensive array growth without wasting too much memory by
	// pre-allocating capacity.
	count := 0
	for _, n := range cache.runners {
		count += len(n.info.Regions)
	}
	return count, nil
}

func (cache *cacheImpl) AssumeRegion(logger klog.Logger, region *corev1.Region) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if _, ok := cache.regionStates[key]; ok {
		return fmt.Errorf("region %v(%v) is in the cache, so can't be assumed", key, klog.KObj(region))
	}

	return cache.addRegion(logger, region, true)
}

func (cache *cacheImpl) FinishBinding(logger klog.Logger, region *corev1.Region) error {
	return cache.finishBinding(logger, region, time.Now())
}

// finishBinding exists to make tests deterministic by injecting now as an argument
func (cache *cacheImpl) finishBinding(logger klog.Logger, region *corev1.Region, now time.Time) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	logger.V(5).Info("Finished binding for region, can be expired", "regionKey", key, "region", klog.KObj(region))
	currState, ok := cache.regionStates[key]
	if ok && cache.assumedRegions.Has(key) {
		if cache.ttl == time.Duration(0) {
			currState.deadline = nil
		} else {
			dl := now.Add(cache.ttl)
			currState.deadline = &dl
		}
		currState.bindingFinished = true
	}
	return nil
}

func (cache *cacheImpl) ForgetRegion(logger klog.Logger, region *corev1.Region) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.regionStates[key]
	_ = currState.deadline
	//if ok && currState.region.Spec.RunnerName != region.Spec.RunnerName {
	//	return fmt.Errorf("region %v(%v) was assumed on %v but assigned to %v", key, klog.KObj(region), region.Spec.RunnerName, currState.region.Spec.RunnerName)
	//}

	// Only assumed region can be forgotten.
	if ok && cache.assumedRegions.Has(key) {
		return cache.removeRegion(logger, region)
	}
	return fmt.Errorf("region %v(%v) wasn't assumed so cannot be forgotten", key, klog.KObj(region))
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) addRegion(logger klog.Logger, region *corev1.Region, assumeRegion bool) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}
	//n, ok := cache.runners[region.Spec.RunnerName]
	//if !ok {
	//	n = newRunnerInfoListItem(framework.NewRunnerInfo())
	//	cache.runners[region.Spec.RunnerName] = n
	//}
	//n.info.AddRegion(region)
	//cache.moveRunnerInfoToHead(logger, region.Spec.RunnerName)
	ps := &regionState{
		region: region,
	}
	cache.regionStates[key] = ps
	if assumeRegion {
		cache.assumedRegions.Insert(key)
	}
	return nil
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) updateRegion(logger klog.Logger, oldRegion, newRegion *corev1.Region) error {
	if err := cache.removeRegion(logger, oldRegion); err != nil {
		return err
	}
	return cache.addRegion(logger, newRegion, false)
}

// Assumes that lock is already acquired.
// Removes a region from the cached runner info. If the runner information was already
// removed and there are no more regions left in the runner, cleans up the runner from
// the cache.
func (cache *cacheImpl) removeRegion(logger klog.Logger, region *corev1.Region) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	//n, ok := cache.runners[region.Spec.RunnerName]
	//if !ok {
	//	logger.Error(nil, "Runner not found when trying to remove region", "runner", klog.KRef("", region.Spec.RunnerName), "regionKey", key, "region", klog.KObj(region))
	//} else {
	//	if err := n.info.RemoveRegion(logger, region); err != nil {
	//		return err
	//	}
	//	if len(n.info.Regions) == 0 && n.info.Runner() == nil {
	//		cache.removeRunnerInfoFromList(logger, region.Spec.RunnerName)
	//	} else {
	//		cache.moveRunnerInfoToHead(logger, region.Spec.RunnerName)
	//	}
	//}

	delete(cache.regionStates, key)
	delete(cache.assumedRegions, key)
	return nil
}

func (cache *cacheImpl) AddRegion(logger klog.Logger, region *corev1.Region) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.regionStates[key]
	switch {
	case ok && cache.assumedRegions.Has(key):
		// When assuming, we've already added the Region to cache,
		// Just update here to make sure the Region's status is up-to-date.
		if err = cache.updateRegion(logger, currState.region, region); err != nil {
			logger.Error(err, "Error occurred while updating region")
		}
		//if currState.region.Spec.RunnerName != region.Spec.RunnerName {
		//	// The region was added to a different runner than it was assumed to.
		//	logger.Info("Region was added to a different runner than it was assumed", "regionKey", key, "region", klog.KObj(region), "assumedRunner", klog.KRef("", region.Spec.RunnerName), "currentRunner", klog.KRef("", currState.region.Spec.RunnerName))
		//	return nil
		//}
	case !ok:
		// Region was expired. We should add it back.
		if err = cache.addRegion(logger, region, false); err != nil {
			logger.Error(err, "Error occurred while adding region")
		}
	default:
		return fmt.Errorf("region %v(%v) was already in added state", key, klog.KObj(region))
	}
	return nil
}

func (cache *cacheImpl) UpdateRegion(logger klog.Logger, oldRegion, newRegion *corev1.Region) error {
	key, err := framework.GetRegionKey(oldRegion)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.regionStates[key]
	if !ok {
		return fmt.Errorf("region %v(%v) is not added to scheduler cache, so cannot be updated", key, klog.KObj(oldRegion))
	}

	// An assumed region won't have Update/Remove event. It needs to have Add event
	// before Update event, in which case the state would change from Assumed to Added.
	if cache.assumedRegions.Has(key) {
		return fmt.Errorf("assumed region %v(%v) should not be updated", key, klog.KObj(oldRegion))
	}

	_ = currState
	//if currState.region.Spec.RunnerName != newRegion.Spec.RunnerName {
	//	logger.Error(nil, "Region updated on a different runner than previously added to", "regionKey", key, "region", klog.KObj(oldRegion))
	//	logger.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	//	klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	//}
	return cache.updateRegion(logger, oldRegion, newRegion)
}

func (cache *cacheImpl) RemoveRegion(logger klog.Logger, region *corev1.Region) error {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.regionStates[key]
	if !ok {
		return fmt.Errorf("region %v(%v) is not found in scheduler cache, so cannot be removed from it", key, klog.KObj(region))
	}
	//if currState.region.Spec.RunnerName != region.Spec.RunnerName {
	//	logger.Error(nil, "Region was added to a different runner than it was assumed", "regionKey", key, "region", klog.KObj(region), "assumedRunner", klog.KRef("", region.Spec.RunnerName), "currentRunner", klog.KRef("", currState.region.Spec.RunnerName))
	//	if region.Spec.RunnerName != "" {
	//		// An empty RunnerName is possible when the scheduler misses a Delete
	//		// event and it gets the last known state from the informer cache.
	//		logger.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	//		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	//	}
	//}
	return cache.removeRegion(logger, currState.region)
}

func (cache *cacheImpl) IsAssumedRegion(region *corev1.Region) (bool, error) {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return false, err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.assumedRegions.Has(key), nil
}

// GetRegion might return a region for which its runner has already been deleted from
// the main cache. This is useful to properly process region update events.
func (cache *cacheImpl) GetRegion(region *corev1.Region) (*corev1.Region, error) {
	key, err := framework.GetRegionKey(region)
	if err != nil {
		return nil, err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	regionState, ok := cache.regionStates[key]
	if !ok {
		return nil, fmt.Errorf("region %v(%v) does not exist in scheduler cache", key, klog.KObj(region))
	}

	return regionState.region, nil
}

func (cache *cacheImpl) AddRunner(logger klog.Logger, runner *corev1.Runner) *framework.RunnerInfo {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.runners[runner.Name]
	if !ok {
		n = newRunnerInfoListItem(framework.NewRunnerInfo())
		cache.runners[runner.Name] = n
	} else {
	}
	cache.moveRunnerInfoToHead(logger, runner.Name)

	cache.runnerTree.addRunner(logger, runner)
	n.info.SetRunner(runner)
	return n.info.Snapshot()
}

func (cache *cacheImpl) UpdateRunner(logger klog.Logger, oldRunner, newRunner *corev1.Runner) *framework.RunnerInfo {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	n, ok := cache.runners[newRunner.Name]
	if !ok {
		n = newRunnerInfoListItem(framework.NewRunnerInfo())
		cache.runners[newRunner.Name] = n
		cache.runnerTree.addRunner(logger, newRunner)
	} else {
	}
	cache.moveRunnerInfoToHead(logger, newRunner.Name)

	cache.runnerTree.updateRunner(logger, oldRunner, newRunner)
	n.info.SetRunner(newRunner)
	return n.info.Snapshot()
}

// RemoveRunner removes a runner from the cache's tree.
// The runner might still have regions because their deletion events didn't arrive
// yet. Those regions are considered removed from the cache, being the runner tree
// the source of truth.
// However, we keep a ghost runner with the list of regions until all region deletion
// events have arrived. A ghost runner is skipped from snapshots.
func (cache *cacheImpl) RemoveRunner(logger klog.Logger, runner *corev1.Runner) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.runners[runner.Name]
	if !ok {
		return fmt.Errorf("runner %v is not found", runner.Name)
	}
	n.info.RemoveRunner()
	// We remove RunnerInfo for this runner only if there aren't any regions on this runner.
	// We can't do it unconditionally, because notifications about regions are delivered
	// in a different watch, and thus can potentially be observed later, even though
	// they happened before runner removal.
	if len(n.info.Regions) == 0 {
		cache.removeRunnerInfoFromList(logger, runner.Name)
	} else {
		cache.moveRunnerInfoToHead(logger, runner.Name)
	}
	if err := cache.runnerTree.removeRunner(logger, runner); err != nil {
		return err
	}
	return nil
}

func (cache *cacheImpl) run(logger klog.Logger) {
	go wait.Until(func() {
		cache.cleanupAssumedRegions(logger, time.Now())
	}, cache.period, cache.stop)
}

// cleanupAssumedRegions exists for making test deterministic by taking time as input argument.
// It also reports metrics on the cache size for runners, regions, and assumed regions.
func (cache *cacheImpl) cleanupAssumedRegions(logger klog.Logger, now time.Time) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	defer cache.updateMetrics()

	// The size of assumedRegions should be small
	for key := range cache.assumedRegions {
		ps, ok := cache.regionStates[key]
		if !ok {
			logger.Error(nil, "Key found in assumed set but not in regionStates, potentially a logical error")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		if !ps.bindingFinished {
			logger.V(5).Info("Could not expire cache for region as binding is still in progress", "regionKey", key, "region", klog.KObj(ps.region))
			continue
		}
		if cache.ttl != 0 && now.After(*ps.deadline) {
			logger.Info("Region expired", "regionKey", key, "region", klog.KObj(ps.region))
			if err := cache.removeRegion(logger, ps.region); err != nil {
				logger.Error(err, "ExpireRegion failed", "regionKey", key, "region", klog.KObj(ps.region))
			}
		}
	}
}

// updateMetrics updates cache size metric values for regions, assumed regions, and runners
func (cache *cacheImpl) updateMetrics() {
	metrics.CacheSize.WithLabelValues("assumed_regions").Set(float64(len(cache.assumedRegions)))
	metrics.CacheSize.WithLabelValues("regions").Set(float64(len(cache.regionStates)))
	metrics.CacheSize.WithLabelValues("runners").Set(float64(len(cache.runners)))
}
