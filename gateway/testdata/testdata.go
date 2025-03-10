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

package testdata

import (
	"embed"
	"testing"

	json "github.com/json-iterator/go"
	"sigs.k8s.io/yaml"

	pb "github.com/olive-io/olive/api/discoverypb"
)

//go:embed rpc.yml
var fs embed.FS

func GetOpenAPIDocs(t *testing.T) *pb.OpenAPI {
	data, err := fs.ReadFile("rpc.yml")
	if err != nil {
		t.Fatal(err)
	}
	data, err = yaml.YAMLToJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	openapi := new(pb.OpenAPI)
	if err = json.Unmarshal(data, openapi); err != nil {
		t.Fatal(err)
	}
	return openapi
}
