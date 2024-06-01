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
	scheme "k8s.io/client-go/kubernetes/scheme"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// The RunnerExpansion interface allows manually adding extra methods to the RunnerInterface.
type RunnerExpansion interface {
	Finalize(ctx context.Context, item *corev1.Runner, opts metav1.UpdateOptions) (*corev1.Runner, error)
}

// UpdateStat was generated because the type contains a Stat member.
func (c *runners) UpdateStat(ctx context.Context, runner *corev1.Runner, opts metav1.UpdateOptions) (result *corev1.Runner, err error) {
	result = &corev1.Runner{}
	err = c.client.Put().
		Resource("runners").
		Name(runner.Name).
		SubResource("stat").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(runner).
		Do(ctx).
		Into(result)
	return
}
