/*
Copyright 2023 The olive Authors

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

package bytesutil

import (
	"reflect"
	"testing"
)

func TestJoin(t *testing.T) {
	type args struct {
		elem [][]byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"join-1", args{elem: [][]byte{[]byte("a")}}, []byte("a")},
		{"join-2", args{elem: [][]byte{[]byte("a"), []byte("b")}}, []byte("a/b")},
		{"join-3", args{elem: [][]byte{[]byte("a"), []byte("b"), []byte("c")}}, []byte("a/b/c")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathJoin(tt.args.elem...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Join() = %v, want %v", got, tt.want)
			}
		})
	}
}
