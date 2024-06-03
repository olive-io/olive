/*
Copyright 2023 The olive Authors

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

package scheduler

import (
	"fmt"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

const (
	DefaultRegionLimit        = 100
	DefaultDefinitionLimit    = 10000
	DefaultInitRegionNum      = 1
	DefaultRegionReplicas     = 3
	DefaultRegionElectionTTL  = 10 // 10s
	DefaultRegionHeartbeatTTL = 1  // 1s
)

type Option func(*Options)

type Options struct {
	RegionLimit        int
	DefinitionLimit    int
	InitRegionNum      int
	RegionReplicas     int
	RegionElectionTTL  int64
	RegionHeartbeatTTL int64
}

func NewOptions(opts ...Option) *Options {
	options := &Options{
		RegionLimit:        DefaultRegionLimit,
		DefinitionLimit:    DefaultDefinitionLimit,
		InitRegionNum:      DefaultInitRegionNum,
		RegionReplicas:     DefaultRegionReplicas,
		RegionElectionTTL:  DefaultRegionElectionTTL,
		RegionHeartbeatTTL: DefaultRegionHeartbeatTTL,
	}

	for _, o := range opts {
		o(options)
	}
	return options
}

func (o *Options) Validate() error {
	var errors []error
	if o.RegionReplicas != 3 && o.RegionReplicas != 5 {
		errors = append(errors, fmt.Errorf("RegionReplicas must be either 3 or 5"))
	}
	if o.InitRegionNum <= 0 {
		errors = append(errors, fmt.Errorf("InitRegionNum must be greater than 0"))
	}
	if o.RegionElectionTTL <= o.RegionHeartbeatTTL {
		errors = append(errors, fmt.Errorf("RegionElectionTTL must be greater than RegionHeartbeatTTL"))
	}
	return utilerrors.NewAggregate(errors)
}

func WithRegionLimit(limit int) Option {
	return func(options *Options) {
		options.RegionLimit = limit
	}
}

func WithDefinitionLimit(limit int) Option {
	return func(options *Options) {
		options.DefinitionLimit = limit
	}
}

func WithInitRegionNum(limit int) Option {
	return func(options *Options) {
		options.InitRegionNum = limit
	}
}

func WithRegionReplicas(replicas int) Option {
	return func(options *Options) {
		options.RegionReplicas = replicas
	}
}

func WithRegionElectionTTL(ttl int64) Option {
	return func(options *Options) {
		options.RegionElectionTTL = ttl
	}
}

func WithRegionHeartbeatTTL(ttl int64) Option {
	return func(options *Options) {
		options.RegionHeartbeatTTL = ttl
	}
}
