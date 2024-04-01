// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, doftware
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"time"

	"go.uber.org/zap"
)

type Options struct {
	Prefix    string
	Namespace string
	Timeout   time.Duration
	Logger    *zap.Logger
}

func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Timeout == 0 {
		options.Timeout = DefaultRegistrarTimeout
	}

	if options.Namespace == "" {
		options.Namespace = DefaultNamespace
	}

	if options.Logger == nil {
		options.Logger = zap.NewNop()
	}

	return options
}

type RegisterOptions struct {
	TTL       time.Duration
	Namespace string
}

type WatchOptions struct {
	// Specify a service to watch
	// If blank, the watch is for all services
	Service   string
	Namespace string
}

type DeregisterOptions struct {
	Namespace string
}

type GetOptions struct {
	Namespace string
}

type ListOptions struct {
	Namespace string
}

type OpenAPIOptions struct {
}

type Option func(*Options)

type RegisterOption func(*RegisterOptions)

type WatchOption func(*WatchOptions)

type DeregisterOption func(*DeregisterOptions)

type GetOption func(*GetOptions)

type ListOption func(*ListOptions)

type OpenAPIOption func(*OpenAPIOptions)

func Prefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

func Namespace(ns string) Option {
	return func(o *Options) {
		o.Namespace = ns
	}
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// SetLogger Specify *zap.Logger
func SetLogger(lg *zap.Logger) Option {
	return func(o *Options) {
		o.Logger = lg
	}
}

func RegisterTTL(t time.Duration) RegisterOption {
	return func(o *RegisterOptions) {
		o.TTL = t
	}
}

func RegisterNamespace(ns string) RegisterOption {
	return func(o *RegisterOptions) {
		o.Namespace = ns
	}
}

func DeregisterNamespace(ns string) DeregisterOption {
	return func(o *DeregisterOptions) {
		o.Namespace = ns
	}
}

// WatchService watches a service
func WatchService(name string) WatchOption {
	return func(o *WatchOptions) {
		o.Service = name
	}
}

func WatchNamespace(ns string) WatchOption {
	return func(o *WatchOptions) {
		o.Namespace = ns
	}
}

func GetNamespace(ns string) GetOption {
	return func(o *GetOptions) {
		o.Namespace = ns
	}
}

func ListNamespace(ns string) ListOption {
	return func(o *ListOptions) {
		o.Namespace = ns
	}
}
