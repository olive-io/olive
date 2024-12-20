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

package runner

import (
	"context"
	urlpkg "net/url"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/warning"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/idutil"
	"github.com/olive-io/olive/pkg/printers"
	printersinternal "github.com/olive-io/olive/pkg/printers/internalversion"
	printerstorage "github.com/olive-io/olive/pkg/printers/storage"
)

// RunnerStorage includes dummy storage for Runner.
type RunnerStorage struct {
	Runner *REST
	Status *StatusREST
}

// NewStorage creates a new RunnerStorage against etcd.
func NewStorage(v3cli *clientv3.Client, optsGetter generic.RESTOptionsGetter, stopCh <-chan struct{}) (RunnerStorage, error) {
	runnerRest, runnerStatusRest, err := NewREST(v3cli, optsGetter, stopCh)
	if err != nil {
		return RunnerStorage{}, err
	}

	return RunnerStorage{
		Runner: runnerRest,
		Status: runnerStatusRest,
	}, nil
}

const (
	defaultRunnerPrefix = "/olive/ring/ids/runner"
)

var deleteOptionWarnings = ""

// REST implements a RESTStorage for runners against etcd
type REST struct {
	*genericregistry.Store

	strategy *runnerStrategy
}

// NewREST returns a RESTStorage object that will work against Runners.
func NewREST(v3cli *clientv3.Client, optsGetter generic.RESTOptionsGetter, stopCh <-chan struct{}) (*REST, *StatusREST, error) {
	ring, err := idutil.NewRing(defaultRunnerPrefix, v3cli)
	if err != nil {
		return nil, nil, err
	}
	ring.Start(stopCh)

	strategy := createStrategy(ring)

	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &corev1.Runner{} },
		NewListFunc:               func() runtime.Object { return &corev1.RunnerList{} },
		PredicateFunc:             MatchRunner,
		DefaultQualifiedResource:  corev1.Resource("runners"),
		SingularQualifiedResource: corev1.Resource("runner"),

		CreateStrategy:      strategy,
		UpdateStrategy:      strategy,
		DeleteStrategy:      strategy,
		ResetFieldsStrategy: strategy,

		TableConvertor: printerstorage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{
		RESTOptions: optsGetter,
		AttrFunc:    GetAttrs,
	}
	if err = store.CompleteWithOptions(options); err != nil {
		return nil, nil, err
	}

	statusStrategy := createStatusStrategy(strategy)

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return &REST{store, strategy}, &StatusREST{store: &statusStore}, nil
}

// Implement CategoriesProvider
var _ rest.CategoriesProvider = &REST{}

// Categories implements the CategoriesProvider interface. Returns a list of categories a resource is part of.
func (r *REST) Categories() []string {
	return []string{"all"}
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	runner := obj.(*corev1.Runner)
	peerURL, err := urlpkg.Parse(runner.Spec.PeerURL)
	if err != nil {
		errorList := field.ErrorList{field.InternalError(field.ToPath(), err)}
		return nil, storage.NewInvalidError(errorList)
	}

	out, _ := r.List(ctx, nil)
	if out != nil {
		for _, item := range out.(*corev1.RunnerList).Items {
			url, err := urlpkg.Parse(item.Spec.PeerURL)
			if err != nil {
				continue
			}
			if url.Host == peerURL.Host {
				return &item, nil
			}
		}
	}
	return r.Store.Create(ctx, obj, createValidation, options)
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	//nolint:staticcheck // SA1019 backwards compatibility
	//nolint: staticcheck
	if options != nil && options.PropagationPolicy == nil && options.OrphanDependents == nil &&
		r.strategy.DefaultGarbageCollectionPolicy(ctx) == rest.OrphanDependents {
		// Throw a warning if delete options are not explicitly set as Runner deletion strategy by default is orphaning
		// pods in v1.
		warning.AddWarning(ctx, "", deleteOptionWarnings)
	}

	runner, err := r.Store.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	if err = r.strategy.PrepareForDelete(ctx, runner); err != nil {
		return nil, false, err
	}
	return r.Store.Delete(ctx, name, deleteValidation, options)
}

func (r *REST) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, deleteOptions *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	if deleteOptions.PropagationPolicy == nil && deleteOptions.OrphanDependents == nil &&
		r.strategy.DefaultGarbageCollectionPolicy(ctx) == rest.OrphanDependents {
		warning.AddWarning(ctx, "", deleteOptionWarnings)
	}
	return r.Store.DeleteCollection(ctx, deleteValidation, deleteOptions, listOptions)
}

// Implement ShortNamesProvider
var _ rest.ShortNamesProvider = &REST{}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"rt"}
}

// StatusREST implements the REST endpoint for changing the status of a resourcequota.
type StatusREST struct {
	store *genericregistry.Store
}

// New creates a new Runner object.
func (r *StatusREST) New() runtime.Object {
	return &corev1.Runner{}
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
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}

func (r *StatusREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.store.ConvertToTable(ctx, object, tableOptions)
}
