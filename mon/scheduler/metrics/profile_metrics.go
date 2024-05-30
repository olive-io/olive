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

package metrics

// This file contains helpers for metrics that are associated to a profile.

var (
	ScheduledResult     = "scheduled"
	UnschedulableResult = "unschedulable"
	ErrorResult         = "error"
)

// DefinitionScheduled can records a successful scheduling attempt and the duration
// since `start`.
func DefinitionScheduled(profile string, duration float64) {
	observeScheduleAttemptAndLatency(ScheduledResult, profile, duration)
}

// DefinitionUnschedulable can records a scheduling attempt for an unschedulable pod
// and the duration since `start`.
func DefinitionUnschedulable(profile string, duration float64) {
	observeScheduleAttemptAndLatency(UnschedulableResult, profile, duration)
}

// DefinitionScheduleError can records a scheduling attempt that had an error and the
// duration since `start`.
func DefinitionScheduleError(profile string, duration float64) {
	observeScheduleAttemptAndLatency(ErrorResult, profile, duration)
}

func observeScheduleAttemptAndLatency(result, profile string, duration float64) {
	schedulingLatency.WithLabelValues(result, profile).Observe(duration)
	scheduleAttempts.WithLabelValues(result, profile).Inc()
}
