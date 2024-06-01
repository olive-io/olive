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

// The NamespaceExpansion interface allows manually adding extra methods to the NamespaceInterface.
type NamespaceExpansion interface {
	Finalize(ctx context.Context, item *corev1.Namespace, opts metav1.UpdateOptions) (*corev1.Namespace, error)
}

// Finalize takes the representation of a namespace to update.  Returns the server's representation of the namespace, and an error, if it occurs.
func (c *namespaces) Finalize(ctx context.Context, namespace *corev1.Namespace, opts metav1.UpdateOptions) (result *corev1.Namespace, err error) {
	result = &corev1.Namespace{}
	err = c.client.Put().Resource("namespaces").Name(namespace.Name).VersionedParams(&opts, scheme.ParameterCodec).SubResource("finalize").Body(namespace).Do(ctx).Into(result)
	return
}
