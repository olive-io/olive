/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/metrics"
)

var (
	cleanAssumedPeriod = 1 * time.Second
)

// New returns a Cache implementation.
// It automatically starts a go routine that manages expiration of assumed definitions.
// "ttl" is how long the assumed definition will get expired.
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
	// a set of assumed definition keys.
	// The key could further be used to get an entry in definitionStates.
	assumedDefinitions sets.Set[string]
	// a map from definition key to definitionState.
	definitionStates map[string]*definitionState
	runners          map[string]*runnerInfoListItem
	// headRunner points to the most recently updated RunnerInfo in "runners". It is the
	// head of the linked list.
	headRunner *runnerInfoListItem
	runnerTree *runnerTree
	// A map from image name to its ImageStateSummary.
	imageStates map[string]*framework.ImageStateSummary
}

type definitionState struct {
	definition *corev1.Definition
	// Used by assumedDefinition to determinate expiration.
	// If deadline is nil, assumedDefinition will never expire.
	deadline *time.Time
	// Used to block cache from expiring assumedDefinition if binding still runs
	bindingFinished bool
}

func newCache(ctx context.Context, ttl, period time.Duration) *cacheImpl {
	logger := klog.FromContext(ctx)
	return &cacheImpl{
		ttl:    ttl,
		period: period,
		stop:   ctx.Done(),

		runners:            make(map[string]*runnerInfoListItem),
		runnerTree:         newRunnerTree(logger, nil),
		assumedDefinitions: sets.New[string](),
		definitionStates:   make(map[string]*definitionState),
		imageStates:        make(map[string]*framework.ImageStateSummary),
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
		Runners:            runners,
		AssumedDefinitions: cache.assumedDefinitions.Union(nil),
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

	// RunnerInfoList and HaveDefinitionsWithAffinityRunnerInfoList must be re-created if a runner was added
	// or removed from the cache.
	updateAllLists := false
	// HaveDefinitionsWithAffinityRunnerInfoList must be re-created if a runner changed its
	// status from having definitions with affinity to NOT having definitions with affinity or the other
	// way around.
	updateRunnersHaveDefinitionsWithAffinity := false
	// HaveDefinitionsWithRequiredAntiAffinityRunnerInfoList must be re-created if a runner changed its
	// status from having definitions with required anti-affinity to NOT having definitions with required
	// anti-affinity or the other way around.
	updateRunnersHaveDefinitionsWithRequiredAntiAffinity := false
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
			// We track runners that have definitions with affinity, here we check if this runner changed its
			// status from having definitions with affinity to NOT having definitions with affinity or the other
			// way around.
			if (len(existing.DefinitionsWithAffinity) > 0) != (len(clone.DefinitionsWithAffinity) > 0) {
				updateRunnersHaveDefinitionsWithAffinity = true
			}
			if (len(existing.DefinitionsWithRequiredAntiAffinity) > 0) != (len(clone.DefinitionsWithRequiredAntiAffinity) > 0) {
				updateRunnersHaveDefinitionsWithRequiredAntiAffinity = true
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

	// Comparing to definitions in runnerTree.
	// Deleted runners get removed from the tree, but they might remain in the runners map
	// if they still have non-deleted Definitions.
	if len(runnerSnapshot.runnerInfoMap) > cache.runnerTree.numRunners {
		cache.removeDeletedRunnersFromSnapshot(runnerSnapshot)
		updateAllLists = true
	}

	if updateAllLists || updateRunnersHaveDefinitionsWithAffinity || updateRunnersHaveDefinitionsWithRequiredAntiAffinity || updateUsedPVCSet {
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
	snapshot.haveDefinitionsWithAffinityRunnerInfoList = make([]*framework.RunnerInfo, 0, cache.runnerTree.numRunners)
	snapshot.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList = make([]*framework.RunnerInfo, 0, cache.runnerTree.numRunners)
	snapshot.usedPVCSet = sets.New[string]()
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
				if len(runnerInfo.DefinitionsWithAffinity) > 0 {
					snapshot.haveDefinitionsWithAffinityRunnerInfoList = append(snapshot.haveDefinitionsWithAffinityRunnerInfoList, runnerInfo)
				}
				if len(runnerInfo.DefinitionsWithRequiredAntiAffinity) > 0 {
					snapshot.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList = append(snapshot.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList, runnerInfo)
				}
				for key := range runnerInfo.PVCRefCounts {
					snapshot.usedPVCSet.Insert(key)
				}
			} else {
				logger.Error(nil, "Runner exists in runnerTree but not in RunnerInfoMap, this should not happen", "runner", klog.KRef("", runnerName))
			}
		}
	} else {
		for _, runnerInfo := range snapshot.runnerInfoList {
			if len(runnerInfo.DefinitionsWithAffinity) > 0 {
				snapshot.haveDefinitionsWithAffinityRunnerInfoList = append(snapshot.haveDefinitionsWithAffinityRunnerInfoList, runnerInfo)
			}
			if len(runnerInfo.DefinitionsWithRequiredAntiAffinity) > 0 {
				snapshot.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList = append(snapshot.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList, runnerInfo)
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

// DefinitionCount returns the number of definitions in the cache (including those from deleted runners).
// DO NOT use outside of tests.
func (cache *cacheImpl) DefinitionCount() (int, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	// definitionFilter is expected to return true for most or all of the definitions. We
	// can avoid expensive array growth without wasting too much memory by
	// pre-allocating capacity.
	count := 0
	for _, n := range cache.runners {
		count += len(n.info.Definitions)
	}
	return count, nil
}

func (cache *cacheImpl) AssumeDefinition(logger klog.Logger, definition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if _, ok := cache.definitionStates[key]; ok {
		return fmt.Errorf("definition %v(%v) is in the cache, so can't be assumed", key, klog.KObj(definition))
	}

	return cache.addDefinition(logger, definition, true)
}

func (cache *cacheImpl) FinishBinding(logger klog.Logger, definition *corev1.Definition) error {
	return cache.finishBinding(logger, definition, time.Now())
}

// finishBinding exists to make tests deterministic by injecting now as an argument
func (cache *cacheImpl) finishBinding(logger klog.Logger, definition *corev1.Definition, now time.Time) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	logger.V(5).Info("Finished binding for definition, can be expired", "definitionKey", key, "definition", klog.KObj(definition))
	currState, ok := cache.definitionStates[key]
	if ok && cache.assumedDefinitions.Has(key) {
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

func (cache *cacheImpl) ForgetDefinition(logger klog.Logger, definition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.definitionStates[key]
	_ = currState.deadline
	//if ok && currState.definition.Spec.RunnerName != definition.Spec.RunnerName {
	//	return fmt.Errorf("definition %v(%v) was assumed on %v but assigned to %v", key, klog.KObj(definition), definition.Spec.RunnerName, currState.definition.Spec.RunnerName)
	//}

	// Only assumed definition can be forgotten.
	if ok && cache.assumedDefinitions.Has(key) {
		return cache.removeDefinition(logger, definition)
	}
	return fmt.Errorf("definition %v(%v) wasn't assumed so cannot be forgotten", key, klog.KObj(definition))
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) addDefinition(logger klog.Logger, definition *corev1.Definition, assumeDefinition bool) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}
	//n, ok := cache.runners[definition.Spec.RunnerName]
	//if !ok {
	//	n = newRunnerInfoListItem(framework.NewRunnerInfo())
	//	cache.runners[definition.Spec.RunnerName] = n
	//}
	//n.info.AddDefinition(definition)
	//cache.moveRunnerInfoToHead(logger, definition.Spec.RunnerName)
	ps := &definitionState{
		definition: definition,
	}
	cache.definitionStates[key] = ps
	if assumeDefinition {
		cache.assumedDefinitions.Insert(key)
	}
	return nil
}

// Assumes that lock is already acquired.
func (cache *cacheImpl) updateDefinition(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error {
	if err := cache.removeDefinition(logger, oldDefinition); err != nil {
		return err
	}
	return cache.addDefinition(logger, newDefinition, false)
}

// Assumes that lock is already acquired.
// Removes a definition from the cached runner info. If the runner information was already
// removed and there are no more definitions left in the runner, cleans up the runner from
// the cache.
func (cache *cacheImpl) removeDefinition(logger klog.Logger, definition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	//n, ok := cache.runners[definition.Spec.RunnerName]
	//if !ok {
	//	logger.Error(nil, "Runner not found when trying to remove definition", "runner", klog.KRef("", definition.Spec.RunnerName), "definitionKey", key, "definition", klog.KObj(definition))
	//} else {
	//	if err := n.info.RemoveDefinition(logger, definition); err != nil {
	//		return err
	//	}
	//	if len(n.info.Definitions) == 0 && n.info.Runner() == nil {
	//		cache.removeRunnerInfoFromList(logger, definition.Spec.RunnerName)
	//	} else {
	//		cache.moveRunnerInfoToHead(logger, definition.Spec.RunnerName)
	//	}
	//}

	delete(cache.definitionStates, key)
	delete(cache.assumedDefinitions, key)
	return nil
}

func (cache *cacheImpl) AddDefinition(logger klog.Logger, definition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.definitionStates[key]
	switch {
	case ok && cache.assumedDefinitions.Has(key):
		// When assuming, we've already added the Definition to cache,
		// Just update here to make sure the Definition's status is up-to-date.
		if err = cache.updateDefinition(logger, currState.definition, definition); err != nil {
			logger.Error(err, "Error occurred while updating definition")
		}
		//if currState.definition.Spec.RunnerName != definition.Spec.RunnerName {
		//	// The definition was added to a different runner than it was assumed to.
		//	logger.Info("Definition was added to a different runner than it was assumed", "definitionKey", key, "definition", klog.KObj(definition), "assumedRunner", klog.KRef("", definition.Spec.RunnerName), "currentRunner", klog.KRef("", currState.definition.Spec.RunnerName))
		//	return nil
		//}
	case !ok:
		// Definition was expired. We should add it back.
		if err = cache.addDefinition(logger, definition, false); err != nil {
			logger.Error(err, "Error occurred while adding definition")
		}
	default:
		return fmt.Errorf("definition %v(%v) was already in added state", key, klog.KObj(definition))
	}
	return nil
}

func (cache *cacheImpl) UpdateDefinition(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(oldDefinition)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.definitionStates[key]
	if !ok {
		return fmt.Errorf("definition %v(%v) is not added to scheduler cache, so cannot be updated", key, klog.KObj(oldDefinition))
	}

	// An assumed definition won't have Update/Remove event. It needs to have Add event
	// before Update event, in which case the state would change from Assumed to Added.
	if cache.assumedDefinitions.Has(key) {
		return fmt.Errorf("assumed definition %v(%v) should not be updated", key, klog.KObj(oldDefinition))
	}

	_ = currState
	//if currState.definition.Spec.RunnerName != newDefinition.Spec.RunnerName {
	//	logger.Error(nil, "Definition updated on a different runner than previously added to", "definitionKey", key, "definition", klog.KObj(oldDefinition))
	//	logger.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	//	klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	//}
	return cache.updateDefinition(logger, oldDefinition, newDefinition)
}

func (cache *cacheImpl) RemoveDefinition(logger klog.Logger, definition *corev1.Definition) error {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.definitionStates[key]
	if !ok {
		return fmt.Errorf("definition %v(%v) is not found in scheduler cache, so cannot be removed from it", key, klog.KObj(definition))
	}
	//if currState.definition.Spec.RunnerName != definition.Spec.RunnerName {
	//	logger.Error(nil, "Definition was added to a different runner than it was assumed", "definitionKey", key, "definition", klog.KObj(definition), "assumedRunner", klog.KRef("", definition.Spec.RunnerName), "currentRunner", klog.KRef("", currState.definition.Spec.RunnerName))
	//	if definition.Spec.RunnerName != "" {
	//		// An empty RunnerName is possible when the scheduler misses a Delete
	//		// event and it gets the last known state from the informer cache.
	//		logger.Error(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	//		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	//	}
	//}
	return cache.removeDefinition(logger, currState.definition)
}

func (cache *cacheImpl) IsAssumedDefinition(definition *corev1.Definition) (bool, error) {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return false, err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.assumedDefinitions.Has(key), nil
}

// GetDefinition might return a definition for which its runner has already been deleted from
// the main cache. This is useful to properly process definition update events.
func (cache *cacheImpl) GetDefinition(definition *corev1.Definition) (*corev1.Definition, error) {
	key, err := framework.GetDefinitionKey(definition)
	if err != nil {
		return nil, err
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	definitionState, ok := cache.definitionStates[key]
	if !ok {
		return nil, fmt.Errorf("definition %v(%v) does not exist in scheduler cache", key, klog.KObj(definition))
	}

	return definitionState.definition, nil
}

func (cache *cacheImpl) AddRunner(logger klog.Logger, runner *monv1.Runner) *framework.RunnerInfo {
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

func (cache *cacheImpl) UpdateRunner(logger klog.Logger, oldRunner, newRunner *monv1.Runner) *framework.RunnerInfo {
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
// The runner might still have definitions because their deletion events didn't arrive
// yet. Those definitions are considered removed from the cache, being the runner tree
// the source of truth.
// However, we keep a ghost runner with the list of definitions until all definition deletion
// events have arrived. A ghost runner is skipped from snapshots.
func (cache *cacheImpl) RemoveRunner(logger klog.Logger, runner *monv1.Runner) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.runners[runner.Name]
	if !ok {
		return fmt.Errorf("runner %v is not found", runner.Name)
	}
	n.info.RemoveRunner()
	// We remove RunnerInfo for this runner only if there aren't any definitions on this runner.
	// We can't do it unconditionally, because notifications about definitions are delivered
	// in a different watch, and thus can potentially be observed later, even though
	// they happened before runner removal.
	if len(n.info.Definitions) == 0 {
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
		cache.cleanupAssumedDefinitions(logger, time.Now())
	}, cache.period, cache.stop)
}

// cleanupAssumedDefinitions exists for making test deterministic by taking time as input argument.
// It also reports metrics on the cache size for runners, definitions, and assumed definitions.
func (cache *cacheImpl) cleanupAssumedDefinitions(logger klog.Logger, now time.Time) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	defer cache.updateMetrics()

	// The size of assumedDefinitions should be small
	for key := range cache.assumedDefinitions {
		ps, ok := cache.definitionStates[key]
		if !ok {
			logger.Error(nil, "Key found in assumed set but not in definitionStates, potentially a logical error")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		if !ps.bindingFinished {
			logger.V(5).Info("Could not expire cache for definition as binding is still in progress", "definitionKey", key, "definition", klog.KObj(ps.definition))
			continue
		}
		if cache.ttl != 0 && now.After(*ps.deadline) {
			logger.Info("Definition expired", "definitionKey", key, "definition", klog.KObj(ps.definition))
			if err := cache.removeDefinition(logger, ps.definition); err != nil {
				logger.Error(err, "ExpireDefinition failed", "definitionKey", key, "definition", klog.KObj(ps.definition))
			}
		}
	}
}

// updateMetrics updates cache size metric values for definitions, assumed definitions, and runners
func (cache *cacheImpl) updateMetrics() {
	metrics.CacheSize.WithLabelValues("assumed_definitions").Set(float64(len(cache.assumedDefinitions)))
	metrics.CacheSize.WithLabelValues("definitions").Set(float64(len(cache.definitionStates)))
	metrics.CacheSize.WithLabelValues("runners").Set(float64(len(cache.runners)))
}
