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

package schedule

type imessage interface {
	payload()
}

// regionAllocMessage allocates a new region on the given olive-runners
type regionAllocMessage struct{}

func (m *regionAllocMessage) payload() {}

// regionExpendMessage expends the capacity of region (maximum is 3)
type regionExpendMessage struct {
	region uint64
}

func (m *regionExpendMessage) payload() {}

// regionMigrateMessage migrates the replicas of region to a new olive-runners
type regionMigrateMessage struct{}

func (m *regionMigrateMessage) payload() {}
