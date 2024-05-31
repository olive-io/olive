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

	clientfeatures "github.com/olive-io/olive/client-go/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

// TestKubeFeaturesRegistered tests that all kube features are registered.
func TestKubeFeaturesRegistered(t *testing.T) {
	registeredFeatures := utilfeature.DefaultFeatureGate.DeepCopy().GetAll()

	for featureName := range defaultOliveFeatureGates {
		if _, ok := registeredFeatures[featureName]; !ok {
			t.Errorf("The feature gate %q is not registered in the DefaultFeatureGate", featureName)
		}
	}
}

// TestClientFeaturesRegistered tests that all client features are registered.
func TestClientFeaturesRegistered(t *testing.T) {
	onlyClientFg := featuregate.NewFeatureGate()
	if err := clientfeatures.AddFeaturesToExistingFeatureGates(&clientAdapter{onlyClientFg}); err != nil {
		t.Fatal(err)
	}
	registeredFeatures := utilfeature.DefaultFeatureGate.DeepCopy().GetAll()

	for featureName := range onlyClientFg.GetAll() {
		if _, ok := registeredFeatures[featureName]; !ok {
			t.Errorf("The client-go's feature gate %q is not registered in the DefaultFeatureGate", featureName)
		}
	}
}

// TestAllRegisteredFeaturesExpected tests that the set of features actually registered does not
// include any features other than those on the list in this package or in client-go's feature
// package.
func TestAllRegisteredFeaturesExpected(t *testing.T) {
	registeredFeatures := utilfeature.DefaultFeatureGate.DeepCopy().GetAll()
	knownFeatureGates := featuregate.NewFeatureGate()
	if err := clientfeatures.AddFeaturesToExistingFeatureGates(&clientAdapter{knownFeatureGates}); err != nil {
		t.Fatal(err)
	}
	if err := knownFeatureGates.Add(defaultOliveFeatureGates); err != nil {
		t.Fatal(err)
	}
	knownFeatures := knownFeatureGates.GetAll()

	for registeredFeature := range registeredFeatures {
		if _, ok := knownFeatures[registeredFeature]; !ok {
			t.Errorf("The feature gate %q is not from known feature gates", registeredFeature)
		}
	}
}
