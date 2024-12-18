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
)

var (
	ErrIsNotPointer = fmt.Errorf("object is not a pointer")
	ErrUnknownGVK   = fmt.Errorf("unknown GroupVersionKind")
	ErrUnknownType  = fmt.Errorf("unknown type")
)

var DefaultScheme = NewScheme()

type DefaultFunc func(src Object, gvk GroupVersionKind) Object

type Scheme struct {
	gvkToTypes map[GroupVersionKind]reflect.Type

	typesToGvk map[reflect.Type]GroupVersionKind

	defaultFuncs map[reflect.Type]DefaultFunc

	gFn DefaultFunc

	observedVersions []GroupVersion
}

// New creates a new Object, and call global DefaultFunc
func (s *Scheme) New(gvk GroupVersionKind) (Object, error) {
	rv, exists := s.gvkToTypes[gvk]
	if !exists {
		return nil, ErrUnknownGVK
	}

	out := reflect.New(rv).Interface().(Object)
	out.GetObjectKind().SetGroupVersionKind(gvk)

	if s.gFn != nil {
		out = s.gFn(out, gvk)
	}

	return out, nil
}

func (s *Scheme) NewList(gvk GroupVersionKind) (Object, error) {
	gvk.Kind += gvk.Kind + "List"
	return s.New(gvk)
}

// IsExists checks GroupVersionKind exists
func (s *Scheme) IsExists(gvk GroupVersionKind) bool {
	_, ok := s.gvkToTypes[gvk]
	return ok
}

// AllGVKs returns all GroupVersionKind
func (s *Scheme) AllGVKs() []GroupVersionKind {
	gvks := make([]GroupVersionKind, 0)
	for gvk, _ := range s.gvkToTypes {
		gvks = append(gvks, gvk)
	}
	return gvks
}

// AllObjects returns all Object
func (s *Scheme) AllObjects() []Object {
	objects := make([]Object, 0)
	for gvk, rv := range s.gvkToTypes {
		out := reflect.New(rv).Interface().(Object)
		out.GetObjectKind().SetGroupVersionKind(gvk)
		objects = append(objects, out)
	}
	return objects
}

// AddKnownTypes add Object to Scheme
func (s *Scheme) AddKnownTypes(gv GroupVersion, types ...Object) error {
	s.addObservedVersion(gv)
	for _, v := range types {
		rt := reflect.TypeOf(v)
		if rt.Kind() != reflect.Ptr {
			return ErrIsNotPointer
		}
		rt = rt.Elem()
		gvk := gv.WithKind(rt.Name())
		s.gvkToTypes[gvk] = rt
		s.typesToGvk[rt] = gvk
	}

	return nil
}

func (s *Scheme) SetTypesGVK(src Object) {
	if !src.GetObjectKind().GroupVersionKind().Empty() {
		return
	}

	rt := reflect.TypeOf(src)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	gvk := s.typesToGvk[rt]
	src.GetObjectKind().SetGroupVersionKind(gvk)
}

// Default call global DefaultFunc specifies DefaultFunc
func (s *Scheme) Default(src Object) Object {
	rt := reflect.TypeOf(src)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	gvk := s.typesToGvk[rt]
	if src.GetObjectKind().GroupVersionKind().Empty() {
		src.GetObjectKind().SetGroupVersionKind(gvk)
	}

	if s.gFn != nil {
		src = s.gFn(src, gvk)
	}

	fn, exists := s.defaultFuncs[rt]
	if exists {
		src = fn(src, gvk)
	}
	return src
}

func (s *Scheme) AddTypeDefaultingFunc(srcType Object, fn DefaultFunc) {
	s.defaultFuncs[reflect.TypeOf(srcType)] = fn
}

func (s *Scheme) AddGlobalDefaultingFunc(fn DefaultFunc) {
	s.gFn = fn
}

func (s *Scheme) addObservedVersion(gv GroupVersion) {
	if gv.Version == "" {
		return
	}

	for _, observedVersion := range s.observedVersions {
		if observedVersion == gv {
			return
		}
	}

	s.observedVersions = append(s.observedVersions, gv)
}

func NewScheme() *Scheme {
	return &Scheme{
		gvkToTypes:       map[GroupVersionKind]reflect.Type{},
		typesToGvk:       map[reflect.Type]GroupVersionKind{},
		defaultFuncs:     map[reflect.Type]DefaultFunc{},
		observedVersions: []GroupVersion{},
	}
}

// NewObject calls DefaultScheme.New()
func NewObject(gvk GroupVersionKind) (Object, error) {
	return DefaultScheme.New(gvk)
}

// IsExists calls DefaultScheme.IsExists()
func IsExists(gvk GroupVersionKind) bool {
	return DefaultScheme.IsExists(gvk)
}

// AllGVKs calls DefaultScheme.AllGVKs()
func AllGVKs() []GroupVersionKind {
	return DefaultScheme.AllGVKs()
}

// AllObjects calls DefaultScheme.AllObjects()
func AllObjects() []Object {
	return DefaultScheme.AllObjects()
}

// AddKnownTypes calls DefaultScheme.AddKnownTypes()
func AddKnownTypes(gv GroupVersion, types ...Object) error {
	return DefaultScheme.AddKnownTypes(gv, types...)
}

// DefaultObject calls DefaultScheme.Default()
func DefaultObject(src Object) Object {
	return DefaultScheme.Default(src)
}

// AddGlobalDefaultingFunc calls DefaultScheme.AddTypeDefaultingFunc()
func AddGlobalDefaultingFunc(fn DefaultFunc) {
	DefaultScheme.AddGlobalDefaultingFunc(fn)
}

// AddTypeDefaultingFunc calls DefaultScheme.AddTypeDefaultingFunc()
func AddTypeDefaultingFunc(srcType Object, fn DefaultFunc) {
	DefaultScheme.AddTypeDefaultingFunc(srcType, fn)
}
