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

package defaultbinder

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/klog/v2/ktesting"

	st "github.com/olive-io/olive/mon/scheduler/testing"

	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
)

func TestDefaultBinder(t *testing.T) {
	testRegion := st.MakeRegion().Name("foo").Namespace("ns").Obj()
	testNode := "foohost.kubernetes.mydomain.com"
	tests := []struct {
		name        string
		injectErr   error
		wantBinding *v1.Binding
	}{
		{
			name: "successful",
			wantBinding: &v1.Binding{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "foo"},
				Target:     v1.ObjectReference{Kind: "Node", Name: testNode},
			},
		}, {
			name:      "binding error",
			injectErr: errors.New("binding error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			var gotBinding *v1.Binding
			client := fake.NewSimpleClientset(testRegion)
			client.PrependReactor("create", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if action.GetSubresource() != "binding" {
					return false, nil, nil
				}
				if tt.injectErr != nil {
					return true, nil, tt.injectErr
				}
				gotBinding = action.(clienttesting.CreateAction).GetObject().(*v1.Binding)
				return true, gotBinding, nil
			})

			fh, err := frameworkruntime.NewFramework(ctx, nil, nil, frameworkruntime.WithClientSet(client))
			if err != nil {
				t.Fatal(err)
			}
			binder := &DefaultBinder{handle: fh}
			status := binder.Bind(ctx, nil, testRegion, testNode)
			if got := status.AsError(); (tt.injectErr != nil) != (got != nil) {
				t.Errorf("got error %q, want %q", got, tt.injectErr)
			}
			if diff := cmp.Diff(tt.wantBinding, gotBinding); diff != "" {
				t.Errorf("got different binding (-want, +got): %s", diff)
			}
		})
	}
}
