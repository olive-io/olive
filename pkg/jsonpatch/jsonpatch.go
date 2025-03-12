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

package jsonpatch

import (
	jsonpatchv5 "github.com/evanphx/json-patch/v5"
	json "github.com/json-iterator/go"
	"github.com/mattbaird/jsonpatch"
)

type Patch struct {
	operations []jsonpatch.JsonPatchOperation
}

func (p *Patch) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(p.operations)
	return data, err
}

func (p *Patch) Each(fn func(string, string, any)) {
	for _, op := range p.operations {
		fn(op.Operation, op.Path, op.Value)
	}
}

func (p *Patch) Len() int {
	return len(p.operations)
}

func CreateJSONPatch(original, target any) (*Patch, error) {
	originalData, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	targetData, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	operations, err := jsonpatch.CreatePatch(originalData, targetData)
	if err != nil {
		return nil, err
	}

	patch := &Patch{
		operations: operations,
	}

	return patch, nil
}

func MargeJSONPatch(original any, patch *Patch) error {
	originalData, err := json.Marshal(original)
	if err != nil {
		return err
	}

	patchData, err := patch.MarshalJSON()
	if err != nil {
		return err
	}
	merge, err := jsonpatchv5.MergePatch(originalData, patchData)
	if err != nil {
		return err
	}

	return json.Unmarshal(merge, original)
}
