// Copyright 2024 The olive Authors
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

package testdata

import (
	"embed"
	"testing"

	json "github.com/json-iterator/go"
	pb "github.com/olive-io/olive/api/discoverypb"
	"sigs.k8s.io/yaml"
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
