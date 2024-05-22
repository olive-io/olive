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

// ConsumerListLister helps list ConsumerLists.
// All objects returned here must be treated as read-only.
type ConsumerListLister interface {
	// List lists all ConsumerLists in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ConsumerList, err error)
	// ConsumerLists returns an object that can list and get ConsumerLists.
	ConsumerLists(namespace string) ConsumerListNamespaceLister
	ConsumerListListerExpansion
}

// consumerListLister implements the ConsumerListLister interface.
type consumerListLister struct {
	indexer cache.Indexer
}

// NewConsumerListLister returns a new ConsumerListLister.
func NewConsumerListLister(indexer cache.Indexer) ConsumerListLister {
	return &consumerListLister{indexer: indexer}
}

// List lists all ConsumerLists in the indexer.
func (s *consumerListLister) List(selector labels.Selector) (ret []*v1.ConsumerList, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ConsumerList))
	})
	return ret, err
}

// ConsumerLists returns an object that can list and get ConsumerLists.
func (s *consumerListLister) ConsumerLists(namespace string) ConsumerListNamespaceLister {
	return consumerListNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ConsumerListNamespaceLister helps list and get ConsumerLists.
// All objects returned here must be treated as read-only.
type ConsumerListNamespaceLister interface {
	// List lists all ConsumerLists in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ConsumerList, err error)
	// Get retrieves the ConsumerList from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.ConsumerList, error)
	ConsumerListNamespaceListerExpansion
}

// consumerListNamespaceLister implements the ConsumerListNamespaceLister
// interface.
type consumerListNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ConsumerLists in the indexer for a given namespace.
func (s consumerListNamespaceLister) List(selector labels.Selector) (ret []*v1.ConsumerList, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ConsumerList))
	})
	return ret, err
}

// Get retrieves the ConsumerList from the indexer for a given namespace and name.
func (s consumerListNamespaceLister) Get(name string) (*v1.ConsumerList, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("consumerlist"), name)
	}
	return obj.(*v1.ConsumerList), nil
}
