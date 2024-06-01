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

	v1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/client-go/generated/clientset/versioned/scheme"
)

// The RunnerExpansion interface allows manually adding extra methods to the RunnerInterface.
type RunnerExpansion interface {
	ApplyStat(ctx context.Context, stat *v1.RunnerStat, opts metav1.UpdateOptions) (*v1.RunnerStat, error)
}

func (c *runners) ApplyStat(ctx context.Context, stat *v1.RunnerStat, opts metav1.UpdateOptions) (result *v1.RunnerStat, err error) {
	result = &v1.RunnerStat{}
	err = c.client.Post().
		Resource("runners").
		Name(stat.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		SubResource("stat").
		Body(stat).
		Do(ctx).
		Into(result)
	return
}
