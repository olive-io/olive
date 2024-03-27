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

package discoverypb

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

func (m *SchemaObject) UnmarshalJSON(data []byte) (err error) {
	result := gjson.Get(string(data), "example")
	if result.Exists() && result.Type != gjson.String {
		text := result.String()
		if result.Type == gjson.Number {
			text = `"` + text + `"`
		}
		patched := []byte(fmt.Sprintf(`{"example": %s}`, text))
		if data, err = jsonpatch.MergePatch(data, patched); err != nil {
			return
		}
	}

	type inner SchemaObject
	var out inner
	if err = json.Unmarshal(data, &out); err != nil {
		return err
	}
	*m = SchemaObject(out)
	return nil
}
