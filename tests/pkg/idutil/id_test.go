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

package idutil

import (
	"context"
	"path"
	"testing"

	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"

	"github.com/olive-io/olive/pkg/idutil"
	ort "github.com/olive-io/olive/pkg/runtime"
)

func TestNewGenerator(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := path.Join(ort.DefaultOlivePrefix, "runner", "id")

	gen, err := idutil.NewGenerator(context.Background(), key, v3client.New(etcd.Server))
	if err != nil {
		t.Fatal(err)
	}

	id := gen.Next()
	id2 := gen.Next()
	if id == id2 {
		t.Errorf("generate the same id %x", id)
	}
}

func BenchmarkNext(b *testing.B) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := path.Join(ort.DefaultOlivePrefix, "runner", "id")

	gen, err := idutil.NewGenerator(context.Background(), key, v3client.New(etcd.Server))
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Next()
	}
}
