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

package gateway

import (
	pb "github.com/olive-io/olive/api/discoverypb"
)

type AddConsumerOptions struct {
	identity *pb.Consumer
	path     string
}

type AddConsumerOption func(*AddConsumerOptions)

func WithConsumerIdentity(identity *pb.Consumer) AddConsumerOption {
	return func(options *AddConsumerOptions) {
		options.identity = identity
	}
}

func WithConsumerActivity(activity pb.ActivityType) AddConsumerOption {
	return func(options *AddConsumerOptions) {
		if options.identity == nil {
			options.identity = &pb.Consumer{}
		}
		options.identity.Activity = activity
	}
}

func WithConsumerId(id string) AddConsumerOption {
	return func(options *AddConsumerOptions) {
		if options.identity == nil {
			options.identity = &pb.Consumer{}
		}
		options.identity.Id = id
	}
}

func WithConsumerAction(action string) AddConsumerOption {
	return func(options *AddConsumerOptions) {
		if options.identity == nil {
			options.identity = &pb.Consumer{}
		}
		options.identity.Action = action
	}
}

func WithConsumerPath(path string) AddConsumerOption {
	return func(options *AddConsumerOptions) {
		options.path = path
	}
}
