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

package storage

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimetesting "k8s.io/apimachinery/pkg/runtime/testing"
	"k8s.io/apimachinery/pkg/test"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/component-base/version"
)

func TestEmulatedStorageVersion(t *testing.T) {
	cases := []struct {
		name             string
		scheme           *runtime.Scheme
		binaryVersion    schema.GroupVersion
		effectiveVersion version.EffectiveVersion
		want             schema.GroupVersion
	}{
		{
			name:             "pick compatible",
			scheme:           AlphaBetaScheme(utilversion.MustParse("1.31"), utilversion.MustParse("1.32")),
			binaryVersion:    v1beta1,
			effectiveVersion: version.NewEffectiveVersion("1.32"),
			want:             v1alpha1,
		},
		{
			name:             "alpha has been replaced, pick binary version",
			scheme:           AlphaReplacedBetaScheme(utilversion.MustParse("1.31"), utilversion.MustParse("1.32")),
			binaryVersion:    v1beta1,
			effectiveVersion: version.NewEffectiveVersion("1.32"),
			want:             v1beta1,
		},
	}

	for _, tc := range cases {
		test.TestScheme()
		t.Run(tc.name, func(t *testing.T) {
			found, err := emulatedStorageVersion(tc.binaryVersion, &CronJob{}, tc.effectiveVersion, tc.scheme)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if found != tc.want {
				t.Errorf("got %v; want %v", found, tc.want)
			}
		})
	}
}

var internalGV = schema.GroupVersion{Group: "workload.example.com", Version: runtime.APIVersionInternal}
var v1alpha1 = schema.GroupVersion{Group: "workload.example.com", Version: "v1alpha1"}
var v1beta1 = schema.GroupVersion{Group: "workload.example.com", Version: "v1beta1"}

type CronJobWithReplacement struct {
	introduced *utilversion.Version
	A          string `json:"A,omitempty"`
	B          int    `json:"B,omitempty"`
}

func (*CronJobWithReplacement) GetObjectKind() schema.ObjectKind { panic("not implemented") }
func (*CronJobWithReplacement) DeepCopyObject() runtime.Object {
	panic("not implemented")
}

func (in *CronJobWithReplacement) APILifecycleIntroduced() (major, minor int) {
	if in.introduced == nil {
		return 0, 0
	}
	return int(in.introduced.Major()), int(in.introduced.Minor())
}

func (in *CronJobWithReplacement) APILifecycleReplacement() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}
}

type CronJob struct {
	introduced *utilversion.Version
	A          string `json:"A,omitempty"`
	B          int    `json:"B,omitempty"`
}

func (*CronJob) GetObjectKind() schema.ObjectKind { panic("not implemented") }
func (*CronJob) DeepCopyObject() runtime.Object {
	panic("not implemented")
}

func (in *CronJob) APILifecycleIntroduced() (major, minor int) {
	if in.introduced == nil {
		return 0, 0
	}
	return int(in.introduced.Major()), int(in.introduced.Minor())
}

func AlphaBetaScheme(alphaVersion, betaVersion *utilversion.Version) *runtime.Scheme {
	s := runtime.NewScheme()
	s.AddKnownTypes(internalGV, &CronJob{})
	s.AddKnownTypes(v1alpha1, &CronJob{introduced: alphaVersion})
	s.AddKnownTypes(v1beta1, &CronJob{introduced: betaVersion})
	s.AddKnownTypeWithName(internalGV.WithKind("CronJob"), &CronJob{})
	s.AddKnownTypeWithName(v1alpha1.WithKind("CronJob"), &CronJob{introduced: alphaVersion})
	s.AddKnownTypeWithName(v1beta1.WithKind("CronJob"), &CronJob{introduced: betaVersion})
	utilruntime.Must(runtimetesting.RegisterConversions(s))
	return s
}

func AlphaReplacedBetaScheme(alphaVersion, betaVersion *utilversion.Version) *runtime.Scheme {
	s := runtime.NewScheme()
	s.AddKnownTypes(internalGV, &CronJob{})
	s.AddKnownTypes(v1alpha1, &CronJobWithReplacement{introduced: alphaVersion})
	s.AddKnownTypes(v1beta1, &CronJob{introduced: betaVersion})
	s.AddKnownTypeWithName(internalGV.WithKind("CronJob"), &CronJob{})
	s.AddKnownTypeWithName(v1alpha1.WithKind("CronJob"), &CronJobWithReplacement{introduced: alphaVersion})
	s.AddKnownTypeWithName(v1beta1.WithKind("CronJob"), &CronJob{introduced: betaVersion})
	utilruntime.Must(runtimetesting.RegisterConversions(s))
	return s
}
