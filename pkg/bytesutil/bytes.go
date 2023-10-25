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

package bytesutil

// PathJoin joins any number of path elements into a single path,
// separating them with slashes. Empty elements are ignored.
// However, if the argument list is empty or all its elements
// are empty, Join returns an empty []byte.
func PathJoin(elem ...[]byte) []byte {
	size := 0
	for _, e := range elem {
		size += len(e)
	}
	if size == 0 {
		return []byte("")
	}
	buf := make([]byte, 0, size+len(elem)-1)
	for _, e := range elem {
		if len(buf) > 0 || e != nil {
			if len(buf) > 0 {
				buf = append(buf, '/')
			}
			buf = append(buf, e...)
		}
	}

	return buf
}
