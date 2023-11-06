// Copyright 2023 Lack (xingyys@gmail.com).
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

package api_test

import (
	"testing"

	"github.com/olive-io/olive/api"
)

// TestInvalidGoTypeIntPanic tests conditions that caused
func TestInvalidGoTypeIntPanic(t *testing.T) {
	result := api.NewLoggablePutRequest(&api.PutRequest{}).String()
	if result != "" {
		t.Errorf("Got result: %s, expected empty string", result)
	}
}
