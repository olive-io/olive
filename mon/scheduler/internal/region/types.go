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

package region

import (
	"sync/atomic"

	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type Snapshot struct {
	region *corev1.Region
	// Sets the limit of definition number in which Region
	DefinitionLimit int
	// the score of Region, calculation by calScore()
	score atomic.Int64
}

func (s *Snapshot) UID() string {
	return s.region.Name
}

func (s *Snapshot) Score() int64 {
	return s.score.Load()
}

func (s *Snapshot) Get() *corev1.Region {
	return s.region.DeepCopy()
}

func (s *Snapshot) Set(r *corev1.Region) {
	s.region = r
}

func (s *Snapshot) DeepCopy() *Snapshot {
	return &Snapshot{
		region:          s.region.DeepCopy(),
		DefinitionLimit: s.DefinitionLimit,
	}
}

func NewSnapshot(region *corev1.Region, limit int) *Snapshot {
	snapshot := &Snapshot{
		region:          region,
		DefinitionLimit: limit,
		score:           atomic.Int64{},
	}
	snapshot.score.Store(snapshot.calScore())
	return snapshot
}

func (s *Snapshot) calScore() int64 {
	region := s.region
	limit := s.DefinitionLimit
	status := region.Status
	score := int64((1-float64(status.Definitions)/float64(limit))*100)%70 +
		int64(status.Replicas*10)%30
	return score
}

type scaleRegion struct {
	region *corev1.Region
}

func newScaleRegion(region *corev1.Region) *scaleRegion {
	return &scaleRegion{region: region}
}

func (s *scaleRegion) UID() string {
	return s.region.Name
}

func (s *scaleRegion) Score() int64 {
	score := int64(s.region.Status.Replicas*10)%60 +
		s.region.Spec.Id*100%40
	return score
}

type regionSelector func(*corev1.Region) bool

type Selector []regionSelector

func NewSelector(options NextOptions) Selector {
	selector := Selector{}
	if options.Name != nil && *options.Name != "" {
		sfn := func(r *corev1.Region) bool { return *options.Name != r.Name }
		selector = append(selector, sfn)
	}
	if len(options.Ignores) > 0 {
		sfn := func(region *corev1.Region) bool {
			idSets := sets.NewString(options.Ignores...)
			return !idSets.Has(region.Name)
		}
		selector = append(selector, sfn)
	}

	return selector
}

func (s Selector) Select(region *corev1.Region) bool {
	for _, item := range s {
		if !item(region) {
			return false
		}
	}
	return true
}

type NextOptions struct {
	// region name
	Name *string
	// ignores the given regions name
	Ignores []string
}

type NextOption func(*NextOptions)

func WithName(name string) NextOption {
	return func(o *NextOptions) {
		o.Name = &name
	}
}

func WithIgnores(names ...string) NextOption {
	return func(o *NextOptions) {
		o.Ignores = names
	}
}
