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
	v1 "github.com/olive-io/olive/apis/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DefinitionLister helps list Definitions.
// All objects returned here must be treated as read-only.
type DefinitionLister interface {
	// List lists all Definitions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Definition, err error)
	// Definitions returns an object that can list and get Definitions.
	Definitions(namespace string) DefinitionNamespaceLister
	DefinitionListerExpansion
}

// definitionLister implements the DefinitionLister interface.
type definitionLister struct {
	indexer cache.Indexer
}

// NewDefinitionLister returns a new DefinitionLister.
func NewDefinitionLister(indexer cache.Indexer) DefinitionLister {
	return &definitionLister{indexer: indexer}
}

// List lists all Definitions in the indexer.
func (s *definitionLister) List(selector labels.Selector) (ret []*v1.Definition, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Definition))
	})
	return ret, err
}

// Definitions returns an object that can list and get Definitions.
func (s *definitionLister) Definitions(namespace string) DefinitionNamespaceLister {
	return definitionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DefinitionNamespaceLister helps list and get Definitions.
// All objects returned here must be treated as read-only.
type DefinitionNamespaceLister interface {
	// List lists all Definitions in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Definition, err error)
	// Get retrieves the Definition from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Definition, error)
	DefinitionNamespaceListerExpansion
}

// definitionNamespaceLister implements the DefinitionNamespaceLister
// interface.
type definitionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Definitions in the indexer for a given namespace.
func (s definitionNamespaceLister) List(selector labels.Selector) (ret []*v1.Definition, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Definition))
	})
	return ret, err
}

// Get retrieves the Definition from the indexer for a given namespace and name.
func (s definitionNamespaceLister) Get(name string) (*v1.Definition, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("definition"), name)
	}
	return obj.(*v1.Definition), nil
}