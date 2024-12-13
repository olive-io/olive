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

package api

import (
	"fmt"
	"reflect"
	"strings"
)

// GroupKind specifies a Group and a kind, but does not force a version. This is useful for identifying
// concepts during lookup stages without having partially valid types
type GroupKind struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
}

func (gk GroupKind) Empty() bool {
	return len(gk.Group) == 0 && len(gk.Kind) == 0
}

func (gk GroupKind) WithVersion(version string) GroupVersionKind {
	return GroupVersionKind{Group: gk.Group, Version: version, Kind: gk.Kind}
}

func (gk GroupKind) String() string {
	if len(gk.Group) == 0 {
		return gk.Kind
	}
	return gk.Kind + "." + gk.Group
}

// GroupVersionKind contains the information of entity, etc Group, Version, Kind
type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// Empty returns true if group, version, and kind is empty
func (gvk GroupVersionKind) Empty() bool {
	return len(gvk.Group) == 0 && len(gvk.Version) == 0 && len(gvk.Kind) == 0
}

func (gvk GroupVersionKind) GroupKind() GroupKind {
	return GroupKind{Group: gvk.Group, Kind: gvk.Kind}
}

func (gvk GroupVersionKind) GroupVersion() GroupVersion {
	return GroupVersion{Group: gvk.Group, Version: gvk.Version}
}

func (gvk GroupVersionKind) APIGroup() string {
	if len(gvk.Group) == 0 {
		return gvk.Version
	}
	return gvk.Group + "/" + gvk.Version
}

func (gvk GroupVersionKind) String() string {
	var s string
	if gvk.Group != "" {
		s += gvk.Group + "/"
	}
	if gvk.Version != "" {
		s += gvk.Version + "."
	}
	return s + gvk.Kind
}

func FromGVK(s string) GroupVersionKind {
	gvk := GroupVersionKind{}
	if idx := strings.Index(s, "/"); idx != -1 {
		gvk.Group = s[:idx]
		s = s[idx+1:]
	}
	if idx := strings.Index(s, "."); idx != -1 {
		gvk.Version = s[:idx]
		s = s[idx+1:]
	} else {
		gvk.Version = "v1"
	}
	gvk.Kind = s
	return gvk
}

type GroupVersion struct {
	Group   string
	Version string
}

// Empty returns true if group and version are empty
func (gv GroupVersion) Empty() bool {
	return len(gv.Group) == 0 && len(gv.Version) == 0
}

// String puts "group" and "version" into a single "group/version" string. For the legacy v1
// it returns "v1"
func (gv GroupVersion) String() string {
	if len(gv.Group) != 0 {
		return gv.Version + "/" + gv.Version
	}
	return gv.Version
}

// ParseGroupVersion turns "group/version" string into a GroupVersion struct. It reports error
// if it cannot parse the string.
func ParseGroupVersion(gv string) (GroupVersion, error) {
	// this can be the internal version for the legacy types
	if (len(gv) == 0) || (gv == "/") {
		return GroupVersion{}, nil
	}

	switch strings.Count(gv, "/") {
	case 0:
		return GroupVersion{"", gv}, nil
	case 1:
		i := strings.Index(gv, "/")
		return GroupVersion{gv[:i], gv[i+1:]}, nil
	default:
		return GroupVersion{}, fmt.Errorf("unexpected GroupVersion string: %v", gv)
	}
}

// WithKind creates GroupVersionKind based on the method receiver's GroupVersion and the passed kind.
func (gv GroupVersion) WithKind(kind string) GroupVersionKind {
	return GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}
}

// WithTypeKind creates a GroupVersionKind based on the method receiver's GroupVersion the passed reflect.Type
func (gv GroupVersion) WithTypeKind(rt reflect.Type) GroupVersionKind {
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: rt.Name()}
}

// WithAnyKind creates a GroupVersionKind based on the method receiver's GroupVersion and the passed interface.
func (gv GroupVersion) WithAnyKind(t any) GroupVersionKind {
	return gv.WithTypeKind(reflect.TypeOf(t))
}
