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

package namespace

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	storageerr "k8s.io/apiserver/pkg/storage/errors"
	"k8s.io/apiserver/pkg/util/dryrun"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	api "github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/printers"
	printersinternal "github.com/olive-io/olive/pkg/printers/internalversion"
	printerstorage "github.com/olive-io/olive/pkg/printers/storage"
)

// REST implements a RESTStorage for namespaces
type REST struct {
	store  *genericregistry.Store
	status *genericregistry.Store
}

// StatusREST implements the REST endpoint for changing the status of a namespace.
type StatusREST struct {
	store *genericregistry.Store
}

// FinalizeREST implements the REST endpoint for finalizing a namespace.
type FinalizeREST struct {
	store *genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against namespaces.
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, *FinalizeREST, error) {
	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &corev1.Namespace{} },
		NewListFunc:               func() runtime.Object { return &corev1.NamespaceList{} },
		PredicateFunc:             MatchNamespace,
		DefaultQualifiedResource:  api.Resource("namespaces"),
		SingularQualifiedResource: api.Resource("namespace"),

		CreateStrategy:      Strategy,
		UpdateStrategy:      Strategy,
		DeleteStrategy:      Strategy,
		ResetFieldsStrategy: Strategy,
		ReturnDeletedObject: true,

		ShouldDeleteDuringUpdate: ShouldDeleteNamespaceDuringUpdate,

		TableConvertor: printerstorage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, nil, nil, err
	}

	statusStore := *store
	statusStore.UpdateStrategy = StatusStrategy
	statusStore.ResetFieldsStrategy = StatusStrategy

	finalizeStore := *store
	finalizeStore.UpdateStrategy = FinalizeStrategy
	finalizeStore.ResetFieldsStrategy = FinalizeStrategy

	return &REST{store: store, status: &statusStore}, &StatusREST{store: &statusStore}, &FinalizeREST{store: &finalizeStore}, nil
}

func (r *REST) NamespaceScoped() bool {
	return r.store.NamespaceScoped()
}

var _ rest.SingularNameProvider = &REST{}

func (r *REST) GetSingularName() string {
	return r.store.GetSingularName()
}

func (r *REST) New() runtime.Object {
	return r.store.New()
}

// Destroy cleans up resources on shutdown.
func (r *REST) Destroy() {
	r.store.Destroy()
}

func (r *REST) NewList() runtime.Object {
	return r.store.NewList()
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.store.List(ctx, options)
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return r.store.Create(ctx, obj, createValidation, options)
}

func (r *REST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return r.store.Watch(ctx, options)
}

// Delete enforces life-cycle rules for namespace termination
func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	nsObj, err := r.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	namespace := nsObj.(*corev1.Namespace)

	// Ensure we have a UID precondition
	if options == nil {
		options = metav1.NewDeleteOptions(0)
	}
	if options.Preconditions == nil {
		options.Preconditions = &metav1.Preconditions{}
	}
	if options.Preconditions.UID == nil {
		options.Preconditions.UID = &namespace.UID
	} else if *options.Preconditions.UID != namespace.UID {
		err = apierrors.NewConflict(
			api.Resource("namespaces"),
			name,
			fmt.Errorf("Precondition failed: UID in precondition: %v, UID in object meta: %v", *options.Preconditions.UID, namespace.UID),
		)
		return nil, false, err
	}
	if options.Preconditions.ResourceVersion != nil && *options.Preconditions.ResourceVersion != namespace.ResourceVersion {
		err = apierrors.NewConflict(
			api.Resource("namespaces"),
			name,
			fmt.Errorf("Precondition failed: ResourceVersion in precondition: %v, ResourceVersion in object meta: %v", *options.Preconditions.ResourceVersion, namespace.ResourceVersion),
		)
		return nil, false, err
	}

	// upon first request to delete, we switch the phase to start namespace termination
	// TODO: enhance graceful deletion's calls to DeleteStrategy to allow phase change and finalizer patterns
	if namespace.DeletionTimestamp.IsZero() {
		key, err := r.store.KeyFunc(ctx, name)
		if err != nil {
			return nil, false, err
		}

		preconditions := storage.Preconditions{UID: options.Preconditions.UID, ResourceVersion: options.Preconditions.ResourceVersion}

		out := r.store.NewFunc()
		err = r.store.Storage.GuaranteedUpdate(
			ctx, key, out, false, &preconditions,
			storage.SimpleUpdate(func(existing runtime.Object) (runtime.Object, error) {
				existingNamespace, ok := existing.(*corev1.Namespace)
				if !ok {
					// wrong type
					return nil, fmt.Errorf("expected *api.Namespace, got %v", existing)
				}
				if err := deleteValidation(ctx, existingNamespace); err != nil {
					return nil, err
				}
				// Set the deletion timestamp if needed
				if existingNamespace.DeletionTimestamp.IsZero() {
					now := metav1.Now()
					existingNamespace.DeletionTimestamp = &now
				}
				// Set the namespace phase to terminating, if needed
				if existingNamespace.Status.Phase != corev1.NamespaceTerminating {
					existingNamespace.Status.Phase = corev1.NamespaceTerminating
				}

				// the current finalizers which are on namespace
				currentFinalizers := map[string]bool{}
				for _, f := range existingNamespace.Finalizers {
					currentFinalizers[f] = true
				}
				// the finalizers we should ensure on namespace
				shouldHaveFinalizers := map[string]bool{
					metav1.FinalizerOrphanDependents: shouldHaveOrphanFinalizer(options, currentFinalizers[metav1.FinalizerOrphanDependents]),
					metav1.FinalizerDeleteDependents: shouldHaveDeleteDependentsFinalizer(options, currentFinalizers[metav1.FinalizerDeleteDependents]),
				}
				// determine whether there are changes
				changeNeeded := false
				for finalizer, shouldHave := range shouldHaveFinalizers {
					changeNeeded = currentFinalizers[finalizer] != shouldHave || changeNeeded
					if shouldHave {
						currentFinalizers[finalizer] = true
					} else {
						delete(currentFinalizers, finalizer)
					}
				}
				// make the changes if needed
				if changeNeeded {
					newFinalizers := []string{}
					for f := range currentFinalizers {
						newFinalizers = append(newFinalizers, f)
					}
					existingNamespace.Finalizers = newFinalizers
				}
				return existingNamespace, nil
			}),
			dryrun.IsDryRun(options.DryRun),
			nil,
		)

		if err != nil {
			err = storageerr.InterpretGetError(err, api.Resource("namespaces"), name)
			err = storageerr.InterpretUpdateError(err, api.Resource("namespaces"), name)
			if _, ok := err.(*apierrors.StatusError); !ok {
				err = apierrors.NewInternalError(err)
			}
			return nil, false, err
		}

		return out, false, nil
	}

	// prior to final deletion, we must ensure that finalizers is empty
	if len(namespace.Spec.Finalizers) != 0 {
		return namespace, false, nil
	}
	return r.store.Delete(ctx, name, deleteValidation, options)
}

// ShouldDeleteNamespaceDuringUpdate adds namespace-specific spec.finalizer checks on top of the default generic ShouldDeleteDuringUpdate behavior
func ShouldDeleteNamespaceDuringUpdate(ctx context.Context, key string, obj, existing runtime.Object) bool {
	ns, ok := obj.(*corev1.Namespace)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected type %T", obj))
		return false
	}
	return len(ns.Spec.Finalizers) == 0 && genericregistry.ShouldDeleteDuringUpdate(ctx, key, obj, existing)
}

func shouldHaveOrphanFinalizer(options *metav1.DeleteOptions, haveOrphanFinalizer bool) bool {
	//nolint:staticcheck // SA1019 backwards compatibility
	if options.OrphanDependents != nil {
		return *options.OrphanDependents
	}
	if options.PropagationPolicy != nil {
		return *options.PropagationPolicy == metav1.DeletePropagationOrphan
	}
	return haveOrphanFinalizer
}

func shouldHaveDeleteDependentsFinalizer(options *metav1.DeleteOptions, haveDeleteDependentsFinalizer bool) bool {
	//nolint:staticcheck // SA1019 backwards compatibility
	if options.OrphanDependents != nil {
		return *options.OrphanDependents == false
	}
	if options.PropagationPolicy != nil {
		return *options.PropagationPolicy == metav1.DeletePropagationForeground
	}
	return haveDeleteDependentsFinalizer
}

func (e *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return e.store.ConvertToTable(ctx, object, tableOptions)
}

// Implement ShortNamesProvider
var _ rest.ShortNamesProvider = &REST{}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"ns"}
}

var _ rest.StorageVersionProvider = &REST{}

func (r *REST) StorageVersion() runtime.GroupVersioner {
	return r.store.StorageVersion()
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *REST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}
func (r *StatusREST) New() runtime.Object {
	return r.store.New()
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

func (r *FinalizeREST) New() runtime.Object {
	return r.store.New()
}

// Destroy cleans up resources on shutdown.
func (r *FinalizeREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Update alters the status finalizers subset of an object.
func (r *FinalizeREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// We are explicitly setting forceAllowCreate to false in the call to the underlying storage because
	// subresources should never allow create on update.
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *FinalizeREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}
