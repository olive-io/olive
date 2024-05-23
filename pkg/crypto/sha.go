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

package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

var defaultSalt = "oliveOo."

type Sha256 struct {
	h hash.Hash
}

func NewSaltSha256(salt string) *Sha256 {
	h := sha256.New()
	h.Write([]byte(salt))
	return &Sha256{h: h}
}

func NewSha256() *Sha256 {
	return NewSaltSha256(defaultSalt)
}

func (s *Sha256) Hash(data []byte) string {
	hashed := s.h.Sum(data)
	return hex.EncodeToString(hashed)
}
