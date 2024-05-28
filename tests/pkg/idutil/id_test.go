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
	"testing"

	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"

	"github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/pkg/idutil"
)

func TestNewIncrement(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := "_olive/ids"

	ctx := context.TODO()
	gen, err := idutil.NewIncrementer(key, v3client.New(etcd.Server), daemon.SetupSignalHandler())
	if err != nil {
		t.Fatal(err)
	}

	id := gen.Next(ctx)
	id2 := gen.Next(ctx)
	if id == id2 {
		t.Errorf("generate the same id %x", id)
	}
}

func BenchmarkIncrementNext(b *testing.B) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := "_olive/ids"

	ctx := context.Background()
	gen, err := idutil.NewIncrementer(key, v3client.New(etcd.Server), daemon.SetupSignalHandler())
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Next(ctx)
	}
}
