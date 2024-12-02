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

package context

import (
	"net/http"
	"testing"

	cxmd "github.com/olive-io/olive/x/context/metadata"
)

func TestRequestToContext(t *testing.T) {
	testData := []struct {
		request *http.Request
		expect  cxmd.Metadata
	}{
		{
			&http.Request{
				Header: http.Header{
					"Foo1": []string{"bar"},
					"Foo2": []string{"bar", "baz"},
				},
			},
			cxmd.Metadata{
				"foo1": "bar",
				"foo2": "bar,baz",
			},
		},
	}

	for _, d := range testData {
		ctx := FromRequest(d.request)
		md, ok := cxmd.FromContext(ctx)
		if !ok {
			t.Fatalf("Expected metadata for request %+v", d.request)
		}
		for k, v := range d.expect {
			if val := md[k]; val != v {
				t.Fatalf("Expected %s for key %s for expected md %+v, got md %+v", v, k, d.expect, md)
			}
		}
	}
}
