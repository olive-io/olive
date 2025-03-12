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

package path

// Join joins any number of path elements into a single path,
// separating them with slashes. Empty elements are ignored.
// However, if the argument list is empty or all its elements
// are empty, Join returns an empty []byte.
func Join(elem ...[]byte) []byte {
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
			if len(buf) > 0 && buf[len(buf)-1] != '/' && e[0] != '/' {
				buf = append(buf, '/')
			}
			buf = append(buf, e...)
		}
	}
	return buf
}
