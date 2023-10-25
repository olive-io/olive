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
			if got := Join(tt.args.elem...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Join() = %v, want %v", got, tt.want)
			}
		})
	}
}
