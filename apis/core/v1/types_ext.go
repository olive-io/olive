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

package v1

func (m *Region) InitialURL() map[int64]string {
	initial := map[int64]string{}
	for _, replica := range m.Spec.InitialReplicas {
		if replica.IsJoin {
			continue
		}
		initial[replica.Id] = replica.RaftAddress
	}
	return initial
}

func (m *Process) NeedScheduler() bool {
	switch m.Status.Phase {
	case ProcessPending, ProcessPrepare:
		return true
	}
	return false
}
