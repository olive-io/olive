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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// BpmnStatApplyConfiguration represents an declarative configuration of the BpmnStat type for use
// with apply.
type BpmnStatApplyConfiguration struct {
	Definitions *int64 `json:"definitions,omitempty"`
	Processes   *int64 `json:"processes,omitempty"`
	Events      *int64 `json:"events,omitempty"`
	Tasks       *int64 `json:"tasks,omitempty"`
}

// BpmnStatApplyConfiguration constructs an declarative configuration of the BpmnStat type for use with
// apply.
func BpmnStat() *BpmnStatApplyConfiguration {
	return &BpmnStatApplyConfiguration{}
}

// WithDefinitions sets the Definitions field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Definitions field is set to the value of the last call.
func (b *BpmnStatApplyConfiguration) WithDefinitions(value int64) *BpmnStatApplyConfiguration {
	b.Definitions = &value
	return b
}

// WithProcesses sets the Processes field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Processes field is set to the value of the last call.
func (b *BpmnStatApplyConfiguration) WithProcesses(value int64) *BpmnStatApplyConfiguration {
	b.Processes = &value
	return b
}

// WithEvents sets the Events field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Events field is set to the value of the last call.
func (b *BpmnStatApplyConfiguration) WithEvents(value int64) *BpmnStatApplyConfiguration {
	b.Events = &value
	return b
}

// WithTasks sets the Tasks field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Tasks field is set to the value of the last call.
func (b *BpmnStatApplyConfiguration) WithTasks(value int64) *BpmnStatApplyConfiguration {
	b.Tasks = &value
	return b
}
