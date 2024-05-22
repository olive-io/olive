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

// RegionReplicaApplyConfiguration represents an declarative configuration of the RegionReplica type for use
// with apply.
type RegionReplicaApplyConfiguration struct {
	Id          *uint64 `json:"id,omitempty"`
	Runner      *uint64 `json:"runner,omitempty"`
	Region      *uint64 `json:"region,omitempty"`
	RaftAddress *string `json:"raftAddress,omitempty"`
	IsNonVoting *bool   `json:"isNonVoting,omitempty"`
	IsWitness   *bool   `json:"isWitness,omitempty"`
	IsJoin      *bool   `json:"isJoin,omitempty"`
}

// RegionReplicaApplyConfiguration constructs an declarative configuration of the RegionReplica type for use with
// apply.
func RegionReplica() *RegionReplicaApplyConfiguration {
	return &RegionReplicaApplyConfiguration{}
}

// WithId sets the Id field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Id field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithId(value uint64) *RegionReplicaApplyConfiguration {
	b.Id = &value
	return b
}

// WithRunner sets the Runner field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Runner field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithRunner(value uint64) *RegionReplicaApplyConfiguration {
	b.Runner = &value
	return b
}

// WithRegion sets the Region field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Region field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithRegion(value uint64) *RegionReplicaApplyConfiguration {
	b.Region = &value
	return b
}

// WithRaftAddress sets the RaftAddress field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RaftAddress field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithRaftAddress(value string) *RegionReplicaApplyConfiguration {
	b.RaftAddress = &value
	return b
}

// WithIsNonVoting sets the IsNonVoting field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IsNonVoting field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithIsNonVoting(value bool) *RegionReplicaApplyConfiguration {
	b.IsNonVoting = &value
	return b
}

// WithIsWitness sets the IsWitness field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IsWitness field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithIsWitness(value bool) *RegionReplicaApplyConfiguration {
	b.IsWitness = &value
	return b
}

// WithIsJoin sets the IsJoin field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IsJoin field is set to the value of the last call.
func (b *RegionReplicaApplyConfiguration) WithIsJoin(value bool) *RegionReplicaApplyConfiguration {
	b.IsJoin = &value
	return b
}
