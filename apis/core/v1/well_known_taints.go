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

const (
	// TaintRunnerNotReady will be added when runner is not ready
	// and removed when runner becomes ready.
	TaintRunnerNotReady = "runner.olive.io/not-ready"

	// TaintRunnerUnreachable will be added when runner becomes unreachable
	// (corresponding to RunnerReady status ConditionUnknown)
	// and removed when runner becomes reachable (RunnerReady status ConditionTrue).
	TaintRunnerUnreachable = "runner.olive.io/unreachable"

	// TaintRunnerUnschedulable will be added when runner becomes unschedulable
	// and removed when runner becomes schedulable.
	TaintRunnerUnschedulable = "runner.olive.io/unschedulable"

	// TaintRunnerMemoryPressure will be added when runner has memory pressure
	// and removed when runner has enough memory.
	TaintRunnerMemoryPressure = "runner.olive.io/memory-pressure"

	// TaintRunnerDiskPressure will be added when runner has disk pressure
	// and removed when runner has enough disk.
	TaintRunnerDiskPressure = "runner.olive.io/disk-pressure"

	// TaintRunnerNetworkUnavailable will be added when runner's network is unavailable
	// and removed when network becomes ready.
	TaintRunnerNetworkUnavailable = "runner.olive.io/network-unavailable"

	// TaintRunnerPIDPressure will be added when runner has pid pressure
	// and removed when runner has enough pid.
	TaintRunnerPIDPressure = "runner.olive.io/pid-pressure"

	// TaintRunnerOutOfService can be added when runner is out of service in case of
	// a non-graceful shutdown
	TaintRunnerOutOfService = "runner.olive.io/out-of-service"
)
