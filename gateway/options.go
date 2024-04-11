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
	dsypb "github.com/olive-io/olive/api/discoverypb"
)

type AddOptions struct {
	identity *dsypb.Consumer
	path     string
}

type AddOption func(*AddOptions)

func AddWithIdentity(identity *dsypb.Consumer) AddOption {
	return func(options *AddOptions) {
		options.identity = identity
	}
}

func AddWithActivity(activity dsypb.ActivityType) AddOption {
	return func(options *AddOptions) {
		if options.identity == nil {
			options.identity = &dsypb.Consumer{}
		}
		options.identity.Activity = activity
	}
}

func AddWithId(id string) AddOption {
	return func(options *AddOptions) {
		if options.identity == nil {
			options.identity = &dsypb.Consumer{}
		}
		options.identity.Id = id
	}
}

func AddWithAction(action string) AddOption {
	return func(options *AddOptions) {
		if options.identity == nil {
			options.identity = &dsypb.Consumer{}
		}
		options.identity.Action = action
	}
}

func AddWithRequest(request any) AddOption {
	return func(options *AddOptions) {
		if options.identity == nil {
			options.identity = &dsypb.Consumer{}
		}
		options.identity.Request = dsypb.BoxFromAny(request)
	}
}

func AddWithResponse(response any) AddOption {
	return func(options *AddOptions) {
		if options.identity == nil {
			options.identity = &dsypb.Consumer{}
		}
		options.identity.Response = dsypb.BoxFromAny(response)
	}
}

func AddWithPath(path string) AddOption {
	return func(options *AddOptions) {
		options.path = path
	}
}
