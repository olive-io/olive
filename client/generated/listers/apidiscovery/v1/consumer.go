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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ConsumerLister helps list Consumers.
// All objects returned here must be treated as read-only.
type ConsumerLister interface {
	// List lists all Consumers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Consumer, err error)
	// Consumers returns an object that can list and get Consumers.
	Consumers(namespace string) ConsumerNamespaceLister
	ConsumerListerExpansion
}

// consumerLister implements the ConsumerLister interface.
type consumerLister struct {
	indexer cache.Indexer
}

// NewConsumerLister returns a new ConsumerLister.
func NewConsumerLister(indexer cache.Indexer) ConsumerLister {
	return &consumerLister{indexer: indexer}
}

// List lists all Consumers in the indexer.
func (s *consumerLister) List(selector labels.Selector) (ret []*v1.Consumer, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Consumer))
	})
	return ret, err
}

// Consumers returns an object that can list and get Consumers.
func (s *consumerLister) Consumers(namespace string) ConsumerNamespaceLister {
	return consumerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ConsumerNamespaceLister helps list and get Consumers.
// All objects returned here must be treated as read-only.
type ConsumerNamespaceLister interface {
	// List lists all Consumers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Consumer, err error)
	// Get retrieves the Consumer from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Consumer, error)
	ConsumerNamespaceListerExpansion
}

// consumerNamespaceLister implements the ConsumerNamespaceLister
// interface.
type consumerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Consumers in the indexer for a given namespace.
func (s consumerNamespaceLister) List(selector labels.Selector) (ret []*v1.Consumer, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Consumer))
	})
	return ret, err
}

// Get retrieves the Consumer from the indexer for a given namespace and name.
func (s consumerNamespaceLister) Get(name string) (*v1.Consumer, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("consumer"), name)
	}
	return obj.(*v1.Consumer), nil
}
