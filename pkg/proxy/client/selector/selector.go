// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package selector is a way to pick a list of service nodes
package selector

import (
	"errors"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrNoneAvailable = errors.New("none available")
)

// ISelector builds on the registry as a mechanism to pick nodes
// and mark their status. This allows host pools and other things
// to be built using various algorithms.
type ISelector interface {
	Init(opts ...Option) error
	Options() Options
	// Select returns a function which should return the next node
	Select(service string, opts ...SelectOption) (Next, error)
	// Mark sets the success/error against a node
	Mark(service string, node *dsypb.Node, err error)
	// Reset returns state back to zero for a service
	Reset(service string)
	// Close renders the selector unusable
	Close() error
}

// Next is a function that returns the next node
// based on the selector's strategy
type Next func() (*dsypb.Node, error)

// Filter is used to filter a service during the selection process
type Filter func([]*dsypb.Service) []*dsypb.Service

// Strategy is a selection strategy e.g random, round robin
type Strategy func([]*dsypb.Service) Next
