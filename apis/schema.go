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

package apis

import (
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = krt.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.CodecFactory{}

	// ParameterCodec handles versioning of objects that are converted to query parameters.
	ParameterCodec krt.ParameterCodec
)

func init() {
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	metav1.AddToGroupVersion(Scheme, unversioned)
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	Codecs = serializer.NewCodecFactory(Scheme)
	ParameterCodec = krt.NewParameterCodec(Scheme)
}

func FromGVK(s string) schema.GroupVersionKind {
	gvk := schema.GroupVersionKind{}
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

type ReflectScheme struct {
	*krt.Scheme
	typesToGvk map[reflect.Type]schema.GroupVersionKind
}

func FromKrtScheme(krt *krt.Scheme) *ReflectScheme {
	rs := &ReflectScheme{
		Scheme:     krt,
		typesToGvk: map[reflect.Type]schema.GroupVersionKind{},
	}
	return rs
}

func NewScheme() *ReflectScheme {
	s := &ReflectScheme{
		Scheme:     krt.NewScheme(),
		typesToGvk: make(map[reflect.Type]schema.GroupVersionKind),
	}
	return s
}

func (s *ReflectScheme) NewList(gvk schema.GroupVersionKind) (krt.Object, error) {
	gvk.Kind += gvk.Kind + "List"
	return s.New(gvk)
}

func (s *ReflectScheme) AddKnownTypes(gv schema.GroupVersion, types ...krt.Object) {
	for _, obj := range types {
		t := reflect.TypeOf(obj)
		if t.Kind() != reflect.Pointer {
			panic("All types must be pointers to structs.")
		}
		t = t.Elem()
		gvk := gv.WithKind(t.Name())
		s.typesToGvk[t] = gvk
		s.Scheme.AddKnownTypes(gv, obj)
	}
}

func (s *ReflectScheme) SetTypesGVK(src krt.Object) {
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
