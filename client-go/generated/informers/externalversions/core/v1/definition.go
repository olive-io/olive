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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	versioned "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	internalinterfaces "github.com/olive-io/olive/client-go/generated/informers/externalversions/internalinterfaces"
	v1 "github.com/olive-io/olive/client-go/generated/listers/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// DefinitionInformer provides access to a shared informer and lister for
// Definitions.
type DefinitionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.DefinitionLister
}

type definitionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewDefinitionInformer constructs a new informer for Definition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDefinitionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredDefinitionInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredDefinitionInformer constructs a new informer for Definition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredDefinitionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1().Definitions(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1().Definitions(namespace).Watch(context.TODO(), options)
			},
		},
		&corev1.Definition{},
		resyncPeriod,
		indexers,
	)
}

func (f *definitionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredDefinitionInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *definitionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1.Definition{}, f.defaultInformer)
}

func (f *definitionInformer) Lister() v1.DefinitionLister {
	return v1.NewDefinitionLister(f.Informer().GetIndexer())
}
