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

package runtime

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// waitingRegionsMap a thread-safe map used to maintain regions waiting in the permit phase.
type waitingRegionsMap struct {
	regions map[types.UID]*waitingRegion
	mu      sync.RWMutex
}

// newWaitingRegionsMap returns a new waitingRegionsMap.
func newWaitingRegionsMap() *waitingRegionsMap {
	return &waitingRegionsMap{
		regions: make(map[types.UID]*waitingRegion),
	}
}

// add a new WaitingRegion to the map.
func (m *waitingRegionsMap) add(wp *waitingRegion) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.regions[wp.GetRegion().UID] = wp
}

// remove a WaitingRegion from the map.
func (m *waitingRegionsMap) remove(uid types.UID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.regions, uid)
}

// get a WaitingRegion from the map.
func (m *waitingRegionsMap) get(uid types.UID) *waitingRegion {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.regions[uid]
}

// iterate acquires a read lock and iterates over the WaitingRegions map.
func (m *waitingRegionsMap) iterate(callback func(framework.WaitingRegion)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.regions {
		callback(v)
	}
}

// waitingRegion represents a region waiting in the permit phase.
type waitingRegion struct {
	region         *corev1.Region
	pendingPlugins map[string]*time.Timer
	s              chan *framework.Status
	mu             sync.RWMutex
}

var _ framework.WaitingRegion = &waitingRegion{}

// newWaitingRegion returns a new waitingRegion instance.
func newWaitingRegion(region *corev1.Region, pluginsMaxWaitTime map[string]time.Duration) *waitingRegion {
	wp := &waitingRegion{
		region: region,
		// Allow() and Reject() calls are non-blocking. This property is guaranteed
		// by using non-blocking send to this channel. This channel has a buffer of size 1
		// to ensure that non-blocking send will not be ignored - possible situation when
		// receiving from this channel happens after non-blocking send.
		s: make(chan *framework.Status, 1),
	}

	wp.pendingPlugins = make(map[string]*time.Timer, len(pluginsMaxWaitTime))
	// The time.AfterFunc calls wp.Reject which iterates through pendingPlugins map. Acquire the
	// lock here so that time.AfterFunc can only execute after newWaitingRegion finishes.
	wp.mu.Lock()
	defer wp.mu.Unlock()
	for k, v := range pluginsMaxWaitTime {
		plugin, waitTime := k, v
		wp.pendingPlugins[plugin] = time.AfterFunc(waitTime, func() {
			msg := fmt.Sprintf("rejected due to timeout after waiting %v at plugin %v",
				waitTime, plugin)
			wp.Reject(plugin, msg)
		})
	}

	return wp
}

// GetRegion returns a reference to the waiting region.
func (w *waitingRegion) GetRegion() *corev1.Region {
	return w.region
}

// GetPendingPlugins returns a list of pending permit plugin's name.
func (w *waitingRegion) GetPendingPlugins() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	plugins := make([]string, 0, len(w.pendingPlugins))
	for p := range w.pendingPlugins {
		plugins = append(plugins, p)
	}

	return plugins
}

// Allow declares the waiting region is allowed to be scheduled by plugin pluginName.
// If this is the last remaining plugin to allow, then a success signal is delivered
// to unblock the region.
func (w *waitingRegion) Allow(pluginName string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if timer, exist := w.pendingPlugins[pluginName]; exist {
		timer.Stop()
		delete(w.pendingPlugins, pluginName)
	}

	// Only signal success status after all plugins have allowed
	if len(w.pendingPlugins) != 0 {
		return
	}

	// The select clause works as a non-blocking send.
	// If there is no receiver, it's a no-op (default case).
	select {
	case w.s <- framework.NewStatus(framework.Success, ""):
	default:
	}
}

// Reject declares the waiting region unschedulable.
func (w *waitingRegion) Reject(pluginName, msg string) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, timer := range w.pendingPlugins {
		timer.Stop()
	}

	// The select clause works as a non-blocking send.
	// If there is no receiver, it's a no-op (default case).
	select {
	case w.s <- framework.NewStatus(framework.Unschedulable, msg).WithPlugin(pluginName):
	default:
	}
}
