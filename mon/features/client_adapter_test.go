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

package features

import (
	"testing"

	"k8s.io/component-base/featuregate"

	clientfeatures "github.com/olive-io/olive/client-go/features"
)

func TestClientAdapterEnabled(t *testing.T) {
	fg := featuregate.NewFeatureGate()
	if err := fg.Add(map[featuregate.Feature]featuregate.FeatureSpec{
		"Foo": {Default: true},
	}); err != nil {
		t.Fatal(err)
	}

	a := &clientAdapter{fg}
	if !a.Enabled("Foo") {
		t.Error("expected Enabled(\"Foo\") to return true")
	}
	var r interface{}
	func() {
		defer func() {
			r = recover()
		}()
		a.Enabled("Bar")
	}()
	if r == nil {
		t.Error("expected Enabled(\"Bar\") to panic due to unknown feature name")
	}
}

func TestClientAdapterAdd(t *testing.T) {
	fg := featuregate.NewFeatureGate()
	a := &clientAdapter{fg}
	defaults := fg.GetAll()
	if err := a.Add(map[clientfeatures.Feature]clientfeatures.FeatureSpec{
		"FeatureAlpha":      {PreRelease: clientfeatures.Alpha, Default: true},
		"FeatureBeta":       {PreRelease: clientfeatures.Beta, Default: false},
		"FeatureGA":         {PreRelease: clientfeatures.GA, Default: true, LockToDefault: true},
		"FeatureDeprecated": {PreRelease: clientfeatures.Deprecated, Default: false, LockToDefault: true},
	}); err != nil {
		t.Fatal(err)
	}
	all := fg.GetAll()
	allexpected := map[featuregate.Feature]featuregate.FeatureSpec{
		"FeatureAlpha":      {PreRelease: featuregate.Alpha, Default: true},
		"FeatureBeta":       {PreRelease: featuregate.Beta, Default: false},
		"FeatureGA":         {PreRelease: featuregate.GA, Default: true, LockToDefault: true},
		"FeatureDeprecated": {PreRelease: featuregate.Deprecated, Default: false, LockToDefault: true},
	}
	for name, spec := range defaults {
		allexpected[name] = spec
	}
	if len(all) != len(allexpected) {
		t.Errorf("expected %d registered features, got %d", len(allexpected), len(all))
	}
	for name, expected := range allexpected {
		actual, ok := all[name]
		if !ok {
			t.Errorf("expected feature %q not found", name)
			continue
		}

		if actual != expected {
			t.Errorf("expected feature %q spec %#v, got spec %#v", name, expected, actual)
		}
	}

	var r interface{}
	func() {
		defer func() {
			r = recover()
		}()
		_ = a.Add(map[clientfeatures.Feature]clientfeatures.FeatureSpec{
			"FeatureAlpha": {PreRelease: "foobar"},
		})
	}()
	if r == nil {
		t.Error("expected panic when adding feature with unknown prerelease")
	}
}
