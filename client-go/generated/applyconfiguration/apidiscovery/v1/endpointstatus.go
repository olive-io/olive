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
	v1 "github.com/olive-io/olive/apis/apidiscovery/v1"
)

// EndpointStatusApplyConfiguration represents an declarative configuration of the EndpointStatus type for use
// with apply.
type EndpointStatusApplyConfiguration struct {
	Phase *v1.EndpointPhase `json:"phase,omitempty"`
}

// EndpointStatusApplyConfiguration constructs an declarative configuration of the EndpointStatus type for use with
// apply.
func EndpointStatus() *EndpointStatusApplyConfiguration {
	return &EndpointStatusApplyConfiguration{}
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *EndpointStatusApplyConfiguration) WithPhase(value v1.EndpointPhase) *EndpointStatusApplyConfiguration {
	b.Phase = &value
	return b
}