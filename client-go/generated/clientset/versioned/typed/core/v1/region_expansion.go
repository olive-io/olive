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

package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/client-go/generated/clientset/versioned/scheme"
)

// The RegionExpansion interface allows manually adding extra methods to the RegionInterface.
type RegionExpansion interface {
	Bind(ctx context.Context, binding *corev1.Binding, opts metav1.CreateOptions) error
}

// Bind applies the provided binding to the named region in the current namespace (binding.Namespace is ignored).
func (c *regions) Bind(ctx context.Context, binding *corev1.Binding, opts metav1.CreateOptions) error {
	return c.client.Post().Namespace("").Resource("regions").Name(binding.Name).VersionedParams(&opts, scheme.ParameterCodec).SubResource("binding").Body(binding).Do(ctx).Error()
}
