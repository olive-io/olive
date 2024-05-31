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

package framework

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
)

// ErrReasonFake is a fake error message denotes the filter function errored.
const ErrReasonFake = "Runners failed the fake plugin"

// FalseFilterPlugin is a filter plugin which always return Unschedulable when Filter function is called.
type FalseFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *FalseFilterPlugin) Name() string {
	return "FalseFilter"
}

// Filter invoked at the filter extension point.
func (pl *FalseFilterPlugin) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	return framework.NewStatus(framework.Unschedulable, ErrReasonFake)
}

// NewFalseFilterPlugin initializes a FalseFilterPlugin and returns it.
func NewFalseFilterPlugin(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &FalseFilterPlugin{}, nil
}

// TrueFilterPlugin is a filter plugin which always return Success when Filter function is called.
type TrueFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *TrueFilterPlugin) Name() string {
	return "TrueFilter"
}

// Filter invoked at the filter extension point.
func (pl *TrueFilterPlugin) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	return nil
}

// NewTrueFilterPlugin initializes a TrueFilterPlugin and returns it.
func NewTrueFilterPlugin(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &TrueFilterPlugin{}, nil
}

type FakePreFilterAndFilterPlugin struct {
	*FakePreFilterPlugin
	*FakeFilterPlugin
}

// Name returns name of the plugin.
func (pl FakePreFilterAndFilterPlugin) Name() string {
	return "FakePreFilterAndFilterPlugin"
}

// FakeFilterPlugin is a test filter plugin to record how many times its Filter() function have
// been called, and it returns different 'Code' depending on its internal 'failedRunnerReturnCodeMap'.
type FakeFilterPlugin struct {
	NumFilterCalled           int32
	FailedRunnerReturnCodeMap map[string]framework.Code
}

// Name returns name of the plugin.
func (pl *FakeFilterPlugin) Name() string {
	return "FakeFilter"
}

// Filter invoked at the filter extension point.
func (pl *FakeFilterPlugin) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	atomic.AddInt32(&pl.NumFilterCalled, 1)

	if returnCode, ok := pl.FailedRunnerReturnCodeMap[runnerInfo.Runner().Name]; ok {
		return framework.NewStatus(returnCode, fmt.Sprintf("injecting failure for region %v", region.Name))
	}

	return nil
}

// NewFakeFilterPlugin initializes a fakeFilterPlugin and returns it.
func NewFakeFilterPlugin(failedRunnerReturnCodeMap map[string]framework.Code) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakeFilterPlugin{
			FailedRunnerReturnCodeMap: failedRunnerReturnCodeMap,
		}, nil
	}
}

// MatchFilterPlugin is a filter plugin which return Success when the evaluated region and runner
// have the same name; otherwise return Unschedulable.
type MatchFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *MatchFilterPlugin) Name() string {
	return "MatchFilter"
}

// Filter invoked at the filter extension point.
func (pl *MatchFilterPlugin) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	runner := runnerInfo.Runner()
	if runner == nil {
		return framework.NewStatus(framework.Error, "runner not found")
	}
	if region.Name == runner.Name {
		return nil
	}
	return framework.NewStatus(framework.Unschedulable, ErrReasonFake)
}

// NewMatchFilterPlugin initializes a MatchFilterPlugin and returns it.
func NewMatchFilterPlugin(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &MatchFilterPlugin{}, nil
}

// FakePreFilterPlugin is a test filter plugin.
type FakePreFilterPlugin struct {
	Result *framework.PreFilterResult
	Status *framework.Status
	name   string
}

// Name returns name of the plugin.
func (pl *FakePreFilterPlugin) Name() string {
	return pl.name
}

// PreFilter invoked at the PreFilter extension point.
func (pl *FakePreFilterPlugin) PreFilter(_ context.Context, _ *framework.CycleState, region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	return pl.Result, pl.Status
}

// PreFilterExtensions no extensions implemented by this plugin.
func (pl *FakePreFilterPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// NewFakePreFilterPlugin initializes a fakePreFilterPlugin and returns it.
func NewFakePreFilterPlugin(name string, result *framework.PreFilterResult, status *framework.Status) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakePreFilterPlugin{
			Result: result,
			Status: status,
			name:   name,
		}, nil
	}
}

// FakeReservePlugin is a test reserve plugin.
type FakeReservePlugin struct {
	Status *framework.Status
}

// Name returns name of the plugin.
func (pl *FakeReservePlugin) Name() string {
	return "FakeReserve"
}

// Reserve invoked at the Reserve extension point.
func (pl *FakeReservePlugin) Reserve(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) *framework.Status {
	return pl.Status
}

// Unreserve invoked at the Unreserve extension point.
func (pl *FakeReservePlugin) Unreserve(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) {
}

// NewFakeReservePlugin initializes a fakeReservePlugin and returns it.
func NewFakeReservePlugin(status *framework.Status) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakeReservePlugin{
			Status: status,
		}, nil
	}
}

// FakePreBindPlugin is a test prebind plugin.
type FakePreBindPlugin struct {
	Status *framework.Status
}

// Name returns name of the plugin.
func (pl *FakePreBindPlugin) Name() string {
	return "FakePreBind"
}

// PreBind invoked at the PreBind extension point.
func (pl *FakePreBindPlugin) PreBind(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) *framework.Status {
	return pl.Status
}

// NewFakePreBindPlugin initializes a fakePreBindPlugin and returns it.
func NewFakePreBindPlugin(status *framework.Status) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakePreBindPlugin{
			Status: status,
		}, nil
	}
}

// FakePermitPlugin is a test permit plugin.
type FakePermitPlugin struct {
	Status  *framework.Status
	Timeout time.Duration
}

// Name returns name of the plugin.
func (pl *FakePermitPlugin) Name() string {
	return "FakePermit"
}

// Permit invoked at the Permit extension point.
func (pl *FakePermitPlugin) Permit(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) (*framework.Status, time.Duration) {
	return pl.Status, pl.Timeout
}

// NewFakePermitPlugin initializes a fakePermitPlugin and returns it.
func NewFakePermitPlugin(status *framework.Status, timeout time.Duration) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakePermitPlugin{
			Status:  status,
			Timeout: timeout,
		}, nil
	}
}

type FakePreScoreAndScorePlugin struct {
	name           string
	score          int64
	preScoreStatus *framework.Status
	scoreStatus    *framework.Status
}

// Name returns name of the plugin.
func (pl *FakePreScoreAndScorePlugin) Name() string {
	return pl.name
}

func (pl *FakePreScoreAndScorePlugin) Score(ctx context.Context, state *framework.CycleState, p *corev1.Region, runnerName string) (int64, *framework.Status) {
	return pl.score, pl.scoreStatus
}

func (pl *FakePreScoreAndScorePlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func (pl *FakePreScoreAndScorePlugin) PreScore(ctx context.Context, state *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	return pl.preScoreStatus
}

func NewFakePreScoreAndScorePlugin(name string, score int64, preScoreStatus, scoreStatus *framework.Status) frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakePreScoreAndScorePlugin{
			name:           name,
			score:          score,
			preScoreStatus: preScoreStatus,
			scoreStatus:    scoreStatus,
		}, nil
	}
}

// NewEqualPrioritizerPlugin returns a factory function to build equalPrioritizerPlugin.
func NewEqualPrioritizerPlugin() frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &FakePreScoreAndScorePlugin{
			name:  "EqualPrioritizerPlugin",
			score: 1,
		}, nil
	}
}
