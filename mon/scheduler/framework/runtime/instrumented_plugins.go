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
	"context"

	compbasemetrics "k8s.io/component-base/metrics"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

type instrumentedFilterPlugin struct {
	framework.FilterPlugin

	metric compbasemetrics.CounterMetric
}

var _ framework.FilterPlugin = &instrumentedFilterPlugin{}

func (p *instrumentedFilterPlugin) Filter(ctx context.Context, state *framework.CycleState, Region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	p.metric.Inc()
	return p.FilterPlugin.Filter(ctx, state, Region, runnerInfo)
}

type instrumentedPreFilterPlugin struct {
	framework.PreFilterPlugin

	metric compbasemetrics.CounterMetric
}

var _ framework.PreFilterPlugin = &instrumentedPreFilterPlugin{}

func (p *instrumentedPreFilterPlugin) PreFilter(ctx context.Context, state *framework.CycleState, Region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	result, status := p.PreFilterPlugin.PreFilter(ctx, state, Region)
	if !status.IsSkip() {
		p.metric.Inc()
	}
	return result, status
}

type instrumentedPreScorePlugin struct {
	framework.PreScorePlugin

	metric compbasemetrics.CounterMetric
}

var _ framework.PreScorePlugin = &instrumentedPreScorePlugin{}

func (p *instrumentedPreScorePlugin) PreScore(ctx context.Context, state *framework.CycleState, Region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	status := p.PreScorePlugin.PreScore(ctx, state, Region, runners)
	if !status.IsSkip() {
		p.metric.Inc()
	}
	return status
}

type instrumentedScorePlugin struct {
	framework.ScorePlugin

	metric compbasemetrics.CounterMetric
}

var _ framework.ScorePlugin = &instrumentedScorePlugin{}

func (p *instrumentedScorePlugin) Score(ctx context.Context, state *framework.CycleState, Region *corev1.Region, runnerName string) (int64, *framework.Status) {
	p.metric.Inc()
	return p.ScorePlugin.Score(ctx, state, Region, runnerName)
}
