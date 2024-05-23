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

package apis

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	apidiscoveryInstall "github.com/olive-io/olive/apis/apidiscovery/install"
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	coreInstall "github.com/olive-io/olive/apis/core/install"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	monInstall "github.com/olive-io/olive/apis/mon/install"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)

	Codec runtime.Codec
)

func init() {
	monInstall.Install(Scheme)
	apidiscoveryInstall.Install(Scheme)
	coreInstall.Install(Scheme)

	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	metav1.AddToGroupVersion(Scheme, unversioned)
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	Codec = Codecs.LegacyCodec(
		unversioned,
		metav1.Unversioned,
		monv1.SchemeGroupVersion,
		apidiscoveryv1.SchemeGroupVersion,
		corev1.SchemeGroupVersion,
	)
}
