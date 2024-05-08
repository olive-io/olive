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

package routes

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/olive-io/olive/client"
)

type AuthGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerAuthGroup() error {
	ag := &AuthGroup{lg: tree.lg, oct: tree.oct}
	summary := ag.Summary()

	group := tree.root.Group("/auth", summary.Name, summary.Description, ag.HandlerChains()...)
	_ = group

	return tree.Group(ag)
}

func (ag *AuthGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "olive.Auth",
		Description: "the documents of olive authentication.",
	}
}

func (ag *AuthGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}
