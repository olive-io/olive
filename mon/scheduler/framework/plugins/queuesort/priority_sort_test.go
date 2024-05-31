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

package queuesort

import (
	"testing"
	"time"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	st "github.com/olive-io/olive/mon/scheduler/testing"
)

func TestLess(t *testing.T) {
	prioritySort := &PrioritySort{}
	var lowPriority, highPriority = int32(10), int32(100)
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	for _, tt := range []struct {
		name     string
		r1       *framework.QueuedRegionInfo
		r2       *framework.QueuedRegionInfo
		expected bool
	}{
		{
			name: "r1.priority less than r2.priority",
			r1: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(lowPriority).Obj()),
			},
			r2: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
			},
			expected: false, // r2 should be ahead of r1 in the queue
		},
		{
			name: "r1.priority greater than r2.priority",
			r1: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
			},
			r2: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(lowPriority).Obj()),
			},
			expected: true, // r1 should be ahead of r2 in the queue
		},
		{
			name: "equal priority. r1 is added to schedulingQ earlier than r2",
			r1: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
				Timestamp:  t1,
			},
			r2: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
				Timestamp:  t2,
			},
			expected: true, // r1 should be ahead of r2 in the queue
		},
		{
			name: "equal priority. r2 is added to schedulingQ earlier than r1",
			r1: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
				Timestamp:  t2,
			},
			r2: &framework.QueuedRegionInfo{
				RegionInfo: mustNewRegionInfo(t, st.MakeRegion().Priority(highPriority).Obj()),
				Timestamp:  t1,
			},
			expected: false, // r2 should be ahead of r1 in the queue
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := prioritySort.Less(tt.r1, tt.r2); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func mustNewRegionInfo(t *testing.T, region *corev1.Region) *framework.RegionInfo {
	podInfo, err := framework.NewRegionInfo(region)
	if err != nil {
		t.Fatal(err)
	}
	return podInfo
}
