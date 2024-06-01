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

package internalversion

import (
	"reflect"
	"sort"
	"testing"

	api "github.com/olive-io/olive/apis/core"
)

func TestSortableResourceNamesSorting(t *testing.T) {
	want := SortableResourceNames{
		api.ResourceName(""),
		api.ResourceName("42"),
		api.ResourceName("bar"),
		api.ResourceName("foo"),
		api.ResourceName("foo"),
		api.ResourceName("foobar"),
	}

	in := SortableResourceNames{
		api.ResourceName("foo"),
		api.ResourceName("42"),
		api.ResourceName("foobar"),
		api.ResourceName("foo"),
		api.ResourceName("bar"),
		api.ResourceName(""),
	}

	sort.Sort(in)
	if !reflect.DeepEqual(in, want) {
		t.Errorf("got %v, want %v", in, want)
	}
}
