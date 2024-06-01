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

package profile

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2/ktesting"

	config "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
)

var fakeRegistry = frameworkruntime.Registry{
	"QueueSort": newFakePlugin("QueueSort"),
	"Bind1":     newFakePlugin("Bind1"),
	"Bind2":     newFakePlugin("Bind2"),
	"Another":   newFakePlugin("Another"),
}

func TestNewMap(t *testing.T) {
	cases := []struct {
		name    string
		cfgs    []config.SchedulerProfile
		wantErr string
	}{
		{
			name: "valid",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "profile-1",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind1"},
							},
						},
					},
				},
				{
					SchedulerName: "profile-2",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind2"},
							},
						},
					},
					PluginConfig: []config.PluginConfig{
						{
							Name: "Bind2",
							Args: runtime.RawExtension{Raw: []byte("{}")},
						},
					},
				},
			},
		},
		{
			name: "different queue sort",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "profile-1",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind1"},
							},
						},
					},
				},
				{
					SchedulerName: "profile-2",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Another"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind2"},
							},
						},
					},
				},
			},
			wantErr: "different queue sort plugins",
		},
		{
			name: "different queue sort args",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "profile-1",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind1"},
							},
						},
					},
					PluginConfig: []config.PluginConfig{
						{
							Name: "QueueSort",
							Args: runtime.RawExtension{Raw: []byte("{}")},
						},
					},
				},
				{
					SchedulerName: "profile-2",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind2"},
							},
						},
					},
				},
			},
			wantErr: "different queue sort plugin args",
		},
		{
			name: "duplicate scheduler name",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "profile-1",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind1"},
							},
						},
					},
				},
				{
					SchedulerName: "profile-1",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind2"},
							},
						},
					},
				},
			},
			wantErr: "duplicate profile",
		},
		{
			name: "scheduler name is needed",
			cfgs: []config.SchedulerProfile{
				{
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
						Bind: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "Bind1"},
							},
						},
					},
				},
			},
			wantErr: "scheduler name is needed",
		},
		{
			name: "plugins required for profile",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "profile-1",
				},
			},
			wantErr: "plugins required for profile",
		},
		{
			name: "invalid framework configuration",
			cfgs: []config.SchedulerProfile{
				{
					SchedulerName: "invalid-profile",
					Plugins: &config.Plugins{
						QueueSort: config.PluginSet{
							Enabled: []config.Plugin{
								{Name: "QueueSort"},
							},
						},
					},
				},
			},
			wantErr: "at least one bind plugin is needed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			m, err := NewMap(ctx, tc.cfgs, fakeRegistry, nilRecorderFactory)
			if err := checkErr(err, tc.wantErr); err != nil {
				t.Fatal(err)
			}
			if len(tc.wantErr) != 0 {
				return
			}
			if len(m) != len(tc.cfgs) {
				t.Errorf("got %d profiles, want %d", len(m), len(tc.cfgs))
			}
		})
	}
}

type fakePlugin struct {
	name string
}

func (p *fakePlugin) Name() string {
	return p.name
}

func (p *fakePlugin) Less(*framework.QueuedRegionInfo, *framework.QueuedRegionInfo) bool {
	return false
}

func (p *fakePlugin) Bind(context.Context, *framework.CycleState, *v1.Pod, string) *framework.Status {
	return nil
}

func newFakePlugin(name string) func(ctx context.Context, object runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &fakePlugin{name: name}, nil
	}
}

func nilRecorderFactory(_ string) events.EventRecorder {
	return nil
}

func checkErr(err error, wantErr string) error {
	if len(wantErr) == 0 {
		return err
	}
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		return fmt.Errorf("got error %q, want %q", err, wantErr)
	}
	return nil
}
