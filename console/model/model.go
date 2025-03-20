/*
Copyright 2025 The olive Authors

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

package model

import (
	"github.com/golang-jwt/jwt/v4"

	"github.com/olive-io/olive/api/types"
)

type ListResult[T any] struct {
	Page int32
	Size int32

	Total int64
	List  []*T
}

func NewListResult[T any](page int32, size int32) *ListResult[T] {
	if page == 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	return &ListResult[T]{Page: page, Size: size, List: make([]*T, 0)}
}

func (r *ListResult[T]) Limit() (int, int) {
	return int((r.Page - 1) * r.Size), int(r.Size)
}

type Model struct {
	ID        uint64 `json:"id" gorm:"primarykey"`
	CreatedAt uint64 `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt uint64 `json:"updatedAt" gorm:"autoUpdateTime"`
}

type Definition struct {
	Model

	DefinitionID int64  `json:"definitionId" gorm:"column:definition_id;index"`
	Version      uint64 `json:"version" gorm:"column:version;index"`
}

type Process struct {
	Model

	ProcessID    int64               `json:"processId" gorm:"column:process_id;index"`
	DefinitionID int64               `json:"definitionId" gorm:"column:definition_id;index"`
	Version      uint64              `json:"version" gorm:"column:version;index"`
	Status       types.ProcessStatus `json:"status" gorm:"column:status"`
}

type WatchRev struct {
	ID uint64 `json:"id" gorm:"column:id;primarykey"`

	Revision int64 `json:"revision" gorm:"column:revision"`
}

type Claims struct {
	jwt.RegisteredClaims
	Id     int64  `json:"id"`
	Secret string `json:"secret"`
}
