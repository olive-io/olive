/*
Copyright 2019 The Kubernetes Authors.

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

package runnername

import (
	"context"
	"reflect"
	"testing"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	st "github.com/olive-io/olive/mon/scheduler/testing"
)

func TestRunnerName(t *testing.T) {
	tests := []struct {
		pod        *corev1.Region
		node       *corev1.Runner
		name       string
		wantStatus *framework.Status
	}{
		{
			pod:  &corev1.Region{},
			node: &corev1.Runner{},
			name: "no host specified",
		},
		{
			pod:  st.MakeRegion().Runners("foo").Obj(),
			node: st.MakeRunner().Name("foo").Obj(),
			name: "host matches",
		},
		{
			pod:        st.MakeRegion().Runners("bar").Obj(),
			node:       st.MakeRunner().Name("foo").Obj(),
			name:       "host doesn't match",
			wantStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReason),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeInfo := framework.NewRunnerInfo()
			nodeInfo.SetRunner(test.node)
			ctx := context.TODO()
			p, err := New(ctx, nil, nil)
			if err != nil {
				t.Fatalf("creating plugin: %v", err)
			}
			gotStatus := p.(framework.FilterPlugin).Filter(ctx, nil, test.pod, nodeInfo)
			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
			}
		})
	}
}
