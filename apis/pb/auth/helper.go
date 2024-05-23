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

package authpbv1

func (m PType) Short() string {
	switch m {
	case PType_POLICY:
		return "p"
	case PType_ROLE:
		return "g"
	default:
		return "null"
	}
}

func (m Resource) Short() string {
	switch m {
	case Resource_MetaMember:
		return "member"
	case Resource_Runner:
		return "runner"
	case Resource_Region:
		return "region"
	case Resource_AuthRole:
		return "role"
	case Resource_AuthUser:
		return "user"
	case Resource_Authentication:
		return "authentication"
	case Resource_BpmnDefinition:
		return "bpmn_definition"
	case Resource_BpmnProcess:
		return "bpmn_process"
	default:
		return "null"
	}
}

func (m Action) Short() string {
	switch m {
	case Action_Read:
		return "read"
	case Action_Write:
		return "write"
	default:
		return "null"
	}
}
