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

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	versioned "github.com/olive-io/olive/client/generated/clientset/versioned"
	internalinterfaces "github.com/olive-io/olive/client/generated/informers/externalversions/internalinterfaces"
	v1 "github.com/olive-io/olive/client/generated/listers/apidiscovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ConsumerInformer provides access to a shared informer and lister for
// Consumers.
type ConsumerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ConsumerLister
}

type consumerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewConsumerInformer constructs a new informer for Consumer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewConsumerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredConsumerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredConsumerInformer constructs a new informer for Consumer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredConsumerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.DiscoveryV1().Consumers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.DiscoveryV1().Consumers(namespace).Watch(context.TODO(), options)
			},
		},
		&apidiscoveryv1.Consumer{},
		resyncPeriod,
		indexers,
	)
}

func (f *consumerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredConsumerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *consumerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apidiscoveryv1.Consumer{}, f.defaultInformer)
}

func (f *consumerInformer) Lister() v1.ConsumerLister {
	return v1.NewConsumerLister(f.Informer().GetIndexer())
}
