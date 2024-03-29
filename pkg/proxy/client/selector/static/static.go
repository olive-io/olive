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

// Package static provides a static resolver which returns the name/ip passed in without any change
package static

import (
	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/proxy/client/selector"
)

// staticSelector is a static selector
type staticSelector struct {
	opts selector.Options
}

func (s *staticSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *staticSelector) Options() selector.Options {
	return s.opts
}

func (s *staticSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	return func() (*dsypb.Node, error) {
		return &dsypb.Node{
			Id:      service,
			Address: service,
		}, nil
	}, nil
}

func (s *staticSelector) Mark(service string, node *dsypb.Node, err error) {
	return
}

func (s *staticSelector) Reset(service string) {
	return
}

func (s *staticSelector) Close() error {
	return nil
}

func NewSelector(opts ...selector.Option) selector.ISelector {
	var options selector.Options
	for _, o := range opts {
		o(&options)
	}
	return &staticSelector{
		opts: options,
	}
}
