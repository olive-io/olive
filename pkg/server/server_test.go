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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGenericServer(t *testing.T) {
	queue := make([]int, 0)

	logger := zap.NewExample()
	s1 := NewEmbedServer(logger)
	s1.Destroy(func() {
		queue = append(queue, 2)
	})
	s1.GoAttach(func() {
		queue = append(queue, 1)
	})
	s1.Shutdown()

	if !assert.Equal(t, queue, []int{1, 2}) {
		return
	}
}
