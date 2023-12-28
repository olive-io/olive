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

package selector

import (
	"context"

	"go.uber.org/zap"

	"github.com/olive-io/olive/pkg/discovery"
)

type Options struct {
	Discovery discovery.IDiscovery
	Strategy  Strategy

	Logger *zap.Logger
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type SelectOptions struct {
	Filters  []Filter
	Strategy Strategy

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Option used to initialise the selector
type Option func(*Options)

// SelectOption used when making a select call
type SelectOption func(*SelectOptions)

// Discovery sets the discovery used by the selector
func Discovery(dsy discovery.IDiscovery) Option {
	return func(o *Options) {
		o.Discovery = dsy
	}
}

// SetStrategy sets the default strategy for the selector
func SetStrategy(fn Strategy) Option {
	return func(o *Options) {
		o.Strategy = fn
	}
}

// SetLogger sets the *zap.Logger for the selector
func SetLogger(lg *zap.Logger) Option {
	return func(o *Options) {
		o.Logger = lg
	}
}

// WithFilter adds a filter function to the list of filters
// used during the Select call.
func WithFilter(fn ...Filter) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, fn...)
	}
}

// WithStrategy sets the selector strategy
func WithStrategy(fn Strategy) SelectOption {
	return func(o *SelectOptions) {
		o.Strategy = fn
	}
}
