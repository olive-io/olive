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

import (
	v1 "github.com/olive-io/olive/apis/core/v1"
)

// ProcessInstanceStatusApplyConfiguration represents an declarative configuration of the ProcessInstanceStatus type for use
// with apply.
type ProcessInstanceStatusApplyConfiguration struct {
	Phase   *v1.ProcessPhase `json:"phase,omitempty"`
	Message *string          `json:"message,omitempty"`
	Region  *uint64          `json:"region,omitempty"`
}

// ProcessInstanceStatusApplyConfiguration constructs an declarative configuration of the ProcessInstanceStatus type for use with
// apply.
func ProcessInstanceStatus() *ProcessInstanceStatusApplyConfiguration {
	return &ProcessInstanceStatusApplyConfiguration{}
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *ProcessInstanceStatusApplyConfiguration) WithPhase(value v1.ProcessPhase) *ProcessInstanceStatusApplyConfiguration {
	b.Phase = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *ProcessInstanceStatusApplyConfiguration) WithMessage(value string) *ProcessInstanceStatusApplyConfiguration {
	b.Message = &value
	return b
}

// WithRegion sets the Region field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Region field is set to the value of the last call.
func (b *ProcessInstanceStatusApplyConfiguration) WithRegion(value uint64) *ProcessInstanceStatusApplyConfiguration {
	b.Region = &value
	return b
}
