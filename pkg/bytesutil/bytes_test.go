// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
