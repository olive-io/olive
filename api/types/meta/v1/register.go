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
	"github.com/olive-io/olive/api"
)

// GroupName is the group name for this API
const GroupName = "meta.olive.io"

// GroupVersion is group version used to register these objects
var GroupVersion = api.GroupVersion{Group: GroupName, Version: "v1"}

var (
	SchemaBuilder = api.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemaBuilder.AddToScheme
	sets          = make([]api.Object, 0)
)

func addKnownTypes(scheme *api.Scheme) error {
	return scheme.AddKnownTypes(GroupVersion, sets...)
}

func init() {
	sets = append(sets)
}
