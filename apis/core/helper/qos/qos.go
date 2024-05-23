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

// NOTE: DO NOT use those helper functions through client-go, the
// package path will be changed in the future.
package qos

import (
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/olive-io/olive/apis/core"
)

var supportedQoSComputeResources = sets.NewString(string(core.ResourceCPU), string(core.ResourceMemory))

func isSupportedQoSComputeResource(name core.ResourceName) bool {
	return supportedQoSComputeResources.Has(string(name))
}
