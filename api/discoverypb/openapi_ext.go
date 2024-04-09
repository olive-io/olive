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
