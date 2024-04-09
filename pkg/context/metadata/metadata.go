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

// Package metadata is a way of defining message headers
package metadata

import (
	"context"
	"strings"
)

type metadataKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

func (md Metadata) Get(key string) (string, bool) {
	// attempt to get as is
	val, ok := md[key]
	if ok {
		return val, ok
	}

	// attempt to get lower case
	val, ok = md[strings.ToLower(key)]
	return val, ok
}

func (md Metadata) Set(key, val string) {
	key = strings.ToLower(key)
	md[key] = val
}

func (md Metadata) Delete(key string) {
	// delete key as-is
	delete(md, key)
	// delete also lower key
	delete(md, strings.ToLower(key))
}

// Copy makes a copy of the metadata
func Copy(md Metadata) Metadata {
	cmd := make(Metadata, len(md))
	for k, v := range md {
		cmd[k] = v
	}
	return cmd
}

// Delete key from metadata
func Delete(ctx context.Context, k string) context.Context {
	return Set(ctx, k, "")
}

// Set add key with val to metadata
func Set(ctx context.Context, k, v string) context.Context {
	md, ok := FromContext(ctx)
	if !ok {
		md = make(Metadata)
	}
	if v == "" {
		delete(md, k)
	} else {
		md[k] = v
	}
	return context.WithValue(ctx, metadataKey{}, md)
}

// Get returns a single value from metadata in the context
func Get(ctx context.Context, key string) (string, bool) {
	md, ok := FromContext(ctx)
	if !ok {
		return "", ok
	}
	// attempt to get as is
	val, ok := md[key]
	if ok {
		return val, ok
	}

	// attempt to get lower case
	val, ok = md[strings.ToLower(key)]

	return val, ok
}

// FromContext returns metadata from the given context
func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metadataKey{}).(Metadata)
	if !ok {
		return nil, ok
	}

	// capitalise all values
	newMD := make(Metadata, len(md))
	for k, v := range md {
		newMD[strings.ToLower(k)] = v
	}

	return newMD, ok
}

// NewContext creates a new context with the given metadata
func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// MergeContext merges metadata to existing metadata, overwriting if specified
func MergeContext(ctx context.Context, patchMd Metadata, overwrite bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := ctx.Value(metadataKey{}).(Metadata)
	cmd := make(Metadata, len(md))
	for k, v := range md {
		cmd[k] = v
	}
	for k, v := range patchMd {
		if _, ok := cmd[k]; ok && !overwrite {
			// skip
		} else if v != "" {
			cmd[k] = v
		} else {
			delete(cmd, k)
		}
	}
	return context.WithValue(ctx, metadataKey{}, cmd)
}
