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

package authpbv1

var (
	ClusterReadScope         = &Scope{Resource: Resource_MetaMember, Action: Action_Read}
	ClusterWriteScope        = &Scope{Resource: Resource_MetaMember, Action: Action_Write}
	RunnerReadScope          = &Scope{Resource: Resource_Runner, Action: Action_Read}
	RunnerWriteScope         = &Scope{Resource: Resource_Runner, Action: Action_Write}
	RegionReadScope          = &Scope{Resource: Resource_Region, Action: Action_Read}
	RegionWriteScope         = &Scope{Resource: Resource_Region, Action: Action_Write}
	RoleReadScope            = &Scope{Resource: Resource_AuthRole, Action: Action_Read}
	RoleWriteScope           = &Scope{Resource: Resource_AuthRole, Action: Action_Write}
	UserReadScope            = &Scope{Resource: Resource_AuthUser, Action: Action_Read}
	UserWriteScope           = &Scope{Resource: Resource_AuthUser, Action: Action_Write}
	AuthReadScope            = &Scope{Resource: Resource_Authentication, Action: Action_Read}
	AuthWriteScope           = &Scope{Resource: Resource_Authentication, Action: Action_Write}
	BpmnDefinitionReadScope  = &Scope{Resource: Resource_BpmnDefinition, Action: Action_Read}
	BpmnDefinitionWriteScope = &Scope{Resource: Resource_BpmnDefinition, Action: Action_Write}
	BpmnProcessReadScope     = &Scope{Resource: Resource_BpmnProcess, Action: Action_Read}
	BpmnProcessWriteScope    = &Scope{Resource: Resource_BpmnProcess, Action: Action_Write}
)

func (m *Scope) Readable() string {
	return m.Resource.Short() + "_" + m.Action.Short()
}
