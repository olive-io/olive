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
