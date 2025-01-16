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

package options

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/olive-io/olive/apis"
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	configv1 "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"

	apidiscoveryInstall "github.com/olive-io/olive/apis/apidiscovery/install"
	configInstall "github.com/olive-io/olive/apis/config/install"
	coreInstall "github.com/olive-io/olive/apis/core/install"
)

var Codec runtime.Codec

func init() {
	apidiscoveryInstall.Install(apis.Scheme)
	configInstall.Install(apis.Scheme)
	coreInstall.Install(apis.Scheme)

	Codec = apis.Codecs.LegacyCodec(
		metav1.Unversioned,
		apidiscoveryv1.SchemeGroupVersion,
		corev1.SchemeGroupVersion,
		core.SchemeGroupVersion,
		configv1.SchemeGroupVersion,
	)
}
