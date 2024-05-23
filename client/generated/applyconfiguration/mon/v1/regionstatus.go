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
	v1 "github.com/olive-io/olive/apis/mon/v1"
)

// RegionStatusApplyConfiguration represents an declarative configuration of the RegionStatus type for use
// with apply.
type RegionStatusApplyConfiguration struct {
	Phase *v1.RegionPhase `json:"phase,omitempty"`
}

// RegionStatusApplyConfiguration constructs an declarative configuration of the RegionStatus type for use with
// apply.
func RegionStatus() *RegionStatusApplyConfiguration {
	return &RegionStatusApplyConfiguration{}
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *RegionStatusApplyConfiguration) WithPhase(value v1.RegionPhase) *RegionStatusApplyConfiguration {
	b.Phase = &value
	return b
}
