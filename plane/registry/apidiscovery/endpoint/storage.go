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

package endpoint

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/warning"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	"github.com/olive-io/olive/pkg/printers"
	printersinternal "github.com/olive-io/olive/pkg/printers/internalversion"
	printerstorage "github.com/olive-io/olive/pkg/printers/storage"
)

// EndpointStorage includes dummy storage for Endpoint.
type EndpointStorage struct {
	Endpoint *REST
	Status   *StatusREST
}

// NewStorage creates a new EndpointStorage against etcd.
func NewStorage(optsGetter generic.RESTOptionsGetter) (EndpointStorage, error) {
	endpointRest, endpointStatusRest, err := NewREST(optsGetter)
	if err != nil {
		return EndpointStorage{}, err
	}

	return EndpointStorage{
		Endpoint: endpointRest,
		Status:   endpointStatusRest,
	}, nil
}

var deleteOptionWarnings = ""

// REST implements a RESTStorage for endpoints against etcd
type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against Endpoints.
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &apidiscoveryv1.Endpoint{} },
		NewListFunc:               func() runtime.Object { return &apidiscoveryv1.EndpointList{} },
		PredicateFunc:             MatchEndpoint,
		DefaultQualifiedResource:  apidiscoveryv1.Resource("endpoints"),
		SingularQualifiedResource: apidiscoveryv1.Resource("endpoint"),

		CreateStrategy:      Strategy,
		UpdateStrategy:      Strategy,
		DeleteStrategy:      Strategy,
		ResetFieldsStrategy: Strategy,

		TableConvertor: printerstorage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, nil, err
	}

	statusStore := *store
	statusStore.UpdateStrategy = StatusStrategy
	statusStore.ResetFieldsStrategy = StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}, nil
}

// Implement CategoriesProvider
var _ rest.CategoriesProvider = &REST{}

// Categories implements the CategoriesProvider interface. Returns a list of categories a resource is part of.
func (r *REST) Categories() []string {
	return []string{"all"}
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	//nolint:staticcheck // SA1019 backwards compatibility
	//nolint: staticcheck
	if options != nil && options.PropagationPolicy == nil && options.OrphanDependents == nil &&
		Strategy.DefaultGarbageCollectionPolicy(ctx) == rest.OrphanDependents {
		// Throw a warning if delete options are not explicitly set as Endpoint deletion strategy by default is orphaning
		// pods in v1.
		warning.AddWarning(ctx, "", deleteOptionWarnings)
	}
	return r.Store.Delete(ctx, name, deleteValidation, options)
}

func (r *REST) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, deleteOptions *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	if deleteOptions.PropagationPolicy == nil && deleteOptions.OrphanDependents == nil &&
		Strategy.DefaultGarbageCollectionPolicy(ctx) == rest.OrphanDependents {
		warning.AddWarning(ctx, "", deleteOptionWarnings)
	}
	return r.Store.DeleteCollection(ctx, deleteValidation, deleteOptions, listOptions)
}

// Implement ShortNamesProvider
var _ rest.ShortNamesProvider = &REST{}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"ep"}
}

// StatusREST implements the REST endpoint for changing the status of a resourcequota.
type StatusREST struct {
	store *genericregistry.Store
}

// New creates a new Endpoint object.
func (r *StatusREST) New() runtime.Object {
	return &apidiscoveryv1.Endpoint{}
}

// Destroy cleans up resources on shutdown.
func (r *StatusREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// We are explicitly setting forceAllowCreate to false in the call to the underlying storage because
	// subresources should never allow create on update.
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}

func (r *StatusREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.store.ConvertToTable(ctx, object, tableOptions)
}
