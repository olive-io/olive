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
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/util/dryrun"
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
	Stat   *StatREST
}

// NewStorage creates a new RunnerStorage against etcd.
func NewStorage(v3cli *clientv3.Client, optsGetter generic.RESTOptionsGetter, stopCh <-chan struct{}) (RunnerStorage, error) {
	runnerRest, runnerStatusRest, runnerStatRest, err := NewREST(v3cli, optsGetter, stopCh)
	if err != nil {
		return RunnerStorage{}, err
	}

	return RunnerStorage{
		Runner: runnerRest,
		Status: runnerStatusRest,
		Stat:   runnerStatRest,
	}, nil
}

const (
	defaultRegionPrefix = "/olive/ring/ids/runner"
)

var deleteOptionWarnings = ""

// REST implements a RESTStorage for runners against etcd
type REST struct {
	*genericregistry.Store

	strategy *runnerStrategy
}

// NewREST returns a RESTStorage object that will work against Runners.
func NewREST(v3cli *clientv3.Client, optsGetter generic.RESTOptionsGetter, stopCh <-chan struct{}) (*REST, *StatusREST, *StatREST, error) {
	ring, err := idutil.NewRing(defaultRegionPrefix, v3cli)
	if err != nil {
		return nil, nil, nil, err
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
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, nil, nil, err
	}

	statusStrategy := createStatusStrategy(strategy)

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	statStrategy := createStatStrategy(strategy)

	statStore := *store
	statStore.NewFunc = func() runtime.Object { return &corev1.RunnerStat{} }
	statStore.CreateStrategy = statStrategy
	statStore.UpdateStrategy = statStrategy
	statStore.ResetFieldsStrategy = statStrategy

	return &REST{store, strategy}, &StatusREST{store: &statusStore}, &StatREST{store: store}, nil
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

var _ = rest.NamedCreater(&StatREST{})
var _ = rest.SubresourceObjectMetaPreserver(&StatREST{})

// StatREST implements the REST endpoint for changing the status of a resourcequota.
type StatREST struct {
	store *genericregistry.Store
}

// Create update the statistic of *corev1.Runner.
func (r *StatREST) Create(ctx context.Context, name string, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (out runtime.Object, err error) {
	runnerStat, ok := obj.(*corev1.RunnerStat)
	if !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("not a RunnerStat object: %#v", obj))
	}

	if name != runnerStat.Name {
		return nil, errors.NewBadRequest("name in URL does not match name in RunnerStat object")
	}

	if createValidation != nil {
		if err = createValidation(ctx, runnerStat.DeepCopyObject()); err != nil {
			return nil, err
		}
	}
	key, err := r.store.KeyFunc(ctx, name)
	if err != nil {
		return nil, err
	}

	statKey := key + "/stat"
	preconditions := &storage.Preconditions{}

	tryUpdate := storage.SimpleUpdate(func(obj runtime.Object) (runtime.Object, error) { return runnerStat, nil })
	isDryRun := dryrun.IsDryRun(options.DryRun)

	err = r.store.Storage.GuaranteedUpdate(ctx, statKey, &corev1.RunnerStat{}, true, preconditions, tryUpdate, isDryRun, nil)

	return runnerStat, err
}

// PreserveRequestObjectMetaSystemFieldsOnSubresourceCreate indicates to a
// handler that this endpoint requires the UID and ResourceVersion to use as
// preconditions. Other fields, such as timestamp, are ignored.
func (r *StatREST) PreserveRequestObjectMetaSystemFieldsOnSubresourceCreate() bool {
	return true
}

// New creates a new Runner object.
func (r *StatREST) New() runtime.Object {
	return &corev1.RunnerStat{}
}

// Destroy cleans up resources on shutdown.
func (r *StatREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	statKey, err := r.store.KeyFunc(ctx, name)
	if err != nil {
		return nil, false, err
	}
	statKey += "/stat"

	var preconditions *storage.Preconditions
	updatePrecondition := objInfo.Preconditions()
	if updatePrecondition != nil {
		preconditions = &storage.Preconditions{
			UID:             updatePrecondition.UID,
			ResourceVersion: updatePrecondition.ResourceVersion,
		}
	}

	var finalStat *corev1.RunnerStat
	err = r.store.Storage.GuaranteedUpdate(ctx, statKey, &corev1.RunnerStat{}, true, preconditions, storage.SimpleUpdate(func(obj runtime.Object) (runtime.Object, error) {
		runnerStat, ok := obj.(*corev1.RunnerStat)
		if !ok {
			return nil, fmt.Errorf("unexpected object: %#v", obj)
		}
		finalStat = runnerStat
		return runnerStat, nil
	}), false, nil)

	return finalStat, true, err
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *StatREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}

func (r *StatREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.store.ConvertToTable(ctx, object, tableOptions)
}
