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

package storage

import (
	"mime"
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/example"
	exampleinstall "k8s.io/apiserver/pkg/apis/example/install"
	examplev1 "k8s.io/apiserver/pkg/apis/example/v1"
	"k8s.io/apiserver/pkg/storage/storagebackend"
)

var (
	v1GroupVersion = schema.GroupVersion{Group: "", Version: "v1"}

	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)
	scheme.AddUnversionedTypes(v1GroupVersion,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	exampleinstall.Install(scheme)
}

type fakeNegotiater struct {
	serializer, streamSerializer runtime.Serializer
	framer                       runtime.Framer
	types, streamTypes           []string
}

func (n *fakeNegotiater) SupportedMediaTypes() []runtime.SerializerInfo {
	var out []runtime.SerializerInfo
	for _, s := range n.types {
		mediaType, _, err := mime.ParseMediaType(s)
		if err != nil {
			panic(err)
		}
		parts := strings.SplitN(mediaType, "/", 2)
		if len(parts) == 1 {
			// this is an error on the server side
			parts = append(parts, "")
		}

		info := runtime.SerializerInfo{
			Serializer:       n.serializer,
			MediaType:        s,
			MediaTypeType:    parts[0],
			MediaTypeSubType: parts[1],
			EncodesAsText:    true,
		}

		for _, t := range n.streamTypes {
			if t == s {
				info.StreamSerializer = &runtime.StreamSerializerInfo{
					EncodesAsText: true,
					Framer:        n.framer,
					Serializer:    n.streamSerializer,
				}
			}
		}
		out = append(out, info)
	}
	return out
}

func (n *fakeNegotiater) UniversalDeserializer() runtime.Decoder {
	return n.serializer
}

func (n *fakeNegotiater) EncoderForVersion(serializer runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return n.serializer
}

func (n *fakeNegotiater) DecoderToVersion(serializer runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return n.serializer
}

func TestConfigurableStorageFactory(t *testing.T) {
	ns := &fakeNegotiater{types: []string{"test/test"}}
	f := NewDefaultStorageFactory(storagebackend.Config{}, "test/test", ns, NewDefaultResourceEncodingConfig(scheme), NewResourceConfig(), nil)
	f.AddCohabitatingResources(example.Resource("test"), schema.GroupResource{Resource: "test2", Group: "2"})
	called := false
	testEncoderChain := func(e runtime.Encoder) runtime.Encoder {
		called = true
		return e
	}
	f.AddSerializationChains(testEncoderChain, nil, example.Resource("test"))
	f.SetEtcdLocation(example.Resource("*"), []string{"/server2"})
	f.SetEtcdPrefix(example.Resource("test"), "/prefix_for_test")

	config, err := f.NewConfig(example.Resource("test"))
	if err != nil {
		t.Fatal(err)
	}
	if config.Prefix != "/prefix_for_test" || !reflect.DeepEqual(config.Transport.ServerList, []string{"/server2"}) {
		t.Errorf("unexpected config %#v", config)
	}
	if !called {
		t.Errorf("expected encoder chain to be called")
	}
}

func TestUpdateEtcdOverrides(t *testing.T) {
	exampleinstall.Install(scheme)

	testCases := []struct {
		resource schema.GroupResource
		servers  []string
	}{
		{
			resource: schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{"http://127.0.0.1:10000"},
		},
		{
			resource: schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{"http://127.0.0.1:10000", "http://127.0.0.1:20000"},
		},
		{
			resource: schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{"http://127.0.0.1:10000"},
		},
	}

	defaultEtcdLocation := []string{"http://127.0.0.1"}
	for i, test := range testCases {
		defaultConfig := storagebackend.Config{
			Prefix: "/registry",
			Transport: storagebackend.TransportConfig{
				ServerList: defaultEtcdLocation,
			},
		}
		storageFactory := NewDefaultStorageFactory(defaultConfig, "", codecs, NewDefaultResourceEncodingConfig(scheme), NewResourceConfig(), nil)
		storageFactory.SetEtcdLocation(test.resource, test.servers)

		var err error
		config, err := storageFactory.NewConfig(test.resource)
		if err != nil {
			t.Errorf("%d: unexpected error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(config.Transport.ServerList, test.servers) {
			t.Errorf("%d: expected %v, got %v", i, test.servers, config.Transport.ServerList)
			continue
		}

		config, err = storageFactory.NewConfig(schema.GroupResource{Group: examplev1.GroupName, Resource: "unlikely"})
		if err != nil {
			t.Errorf("%d: unexpected error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(config.Transport.ServerList, defaultEtcdLocation) {
			t.Errorf("%d: expected %v, got %v", i, defaultEtcdLocation, config.Transport.ServerList)
			continue
		}

	}
}

func TestConfigs(t *testing.T) {
	exampleinstall.Install(scheme)
	defaultEtcdLocations := []string{"http://127.0.0.1", "http://127.0.0.2"}

	testCases := []struct {
		resource    *schema.GroupResource
		servers     []string
		wantConfigs []storagebackend.Config
	}{
		{
			wantConfigs: []storagebackend.Config{
				{Transport: storagebackend.TransportConfig{ServerList: defaultEtcdLocations}, Prefix: "/registry"},
			},
		},
		{
			resource: &schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{},
			wantConfigs: []storagebackend.Config{
				{Transport: storagebackend.TransportConfig{ServerList: defaultEtcdLocations}, Prefix: "/registry"},
			},
		},
		{
			resource: &schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{"http://127.0.0.1:10000"},
			wantConfigs: []storagebackend.Config{
				{Transport: storagebackend.TransportConfig{ServerList: defaultEtcdLocations}, Prefix: "/registry"},
				{Transport: storagebackend.TransportConfig{ServerList: []string{"http://127.0.0.1:10000"}}, Prefix: "/registry"},
			},
		},
		{
			resource: &schema.GroupResource{Group: example.GroupName, Resource: "resource"},
			servers:  []string{"http://127.0.0.1:10000", "https://127.0.0.1", "http://127.0.0.2"},
			wantConfigs: []storagebackend.Config{
				{Transport: storagebackend.TransportConfig{ServerList: defaultEtcdLocations}, Prefix: "/registry"},
				{Transport: storagebackend.TransportConfig{ServerList: []string{"http://127.0.0.1:10000", "https://127.0.0.1", "http://127.0.0.2"}}, Prefix: "/registry"},
			},
		},
	}

	for i, test := range testCases {
		defaultConfig := storagebackend.Config{
			Prefix: "/registry",
			Transport: storagebackend.TransportConfig{
				ServerList: defaultEtcdLocations,
			},
		}
		storageFactory := NewDefaultStorageFactory(defaultConfig, "", codecs, NewDefaultResourceEncodingConfig(scheme), NewResourceConfig(), nil)
		if test.resource != nil {
			storageFactory.SetEtcdLocation(*test.resource, test.servers)
		}

		got := storageFactory.Configs()
		if !reflect.DeepEqual(test.wantConfigs, got) {
			t.Errorf("%d: expected %v, got %v", i, test.wantConfigs, got)
			continue
		}
	}
}
