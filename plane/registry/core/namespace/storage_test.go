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
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistrytest "k8s.io/apiserver/pkg/registry/generic/testing"
	"k8s.io/apiserver/pkg/registry/rest"
	etcd3testing "k8s.io/apiserver/pkg/storage/etcd3/testing"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/plane/registry/registrytest"
)

func newStorage(t *testing.T) (*REST, *etcd3testing.EtcdTestServer) {
	etcdStorage, server := registrytest.NewEtcdStorage(t, "")
	restOptions := generic.RESTOptions{StorageConfig: etcdStorage, Decorator: generic.UndecoratedStorage, DeleteCollectionWorkers: 1, ResourcePrefix: "namespaces"}
	namespaceStorage, _, _, err := NewREST(restOptions)
	if err != nil {
		t.Fatalf("unexpected error from REST storage: %v", err)
	}
	return namespaceStorage, server
}

func validNewNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
}

func TestCreate(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	test := genericregistrytest.New(t, storage.store).ClusterScope()
	namespace := validNewNamespace()
	namespace.ObjectMeta = metav1.ObjectMeta{GenerateName: "foo"}
	test.TestCreate(
		// valid
		namespace,
		// invalid
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "bad value"},
		},
	)
}

func TestCreateSetsFields(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	namespace := validNewNamespace()
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	_, err := storage.Create(ctx, namespace, rest.ValidateAllObjectFunc, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	object, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	actual := object.(*corev1.Namespace)
	if actual.Name != namespace.Name {
		t.Errorf("unexpected namespace: %#v", actual)
	}
	if len(actual.UID) == 0 {
		t.Errorf("expected namespace UID to be set: %#v", actual)
	}
	if actual.Status.Phase != corev1.NamespaceActive {
		t.Errorf("expected namespace phase to be set to active, but %v", actual.Status.Phase)
	}
}

func TestDelete(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	test := genericregistrytest.New(t, storage.store).ClusterScope().ReturnDeletedObject()
	test.TestDelete(validNewNamespace())
}

func TestGet(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	test := genericregistrytest.New(t, storage.store).ClusterScope()

	// note that this ultimately may call validation
	test.TestGet(validNewNamespace())
}

func TestList(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	test := genericregistrytest.New(t, storage.store).ClusterScope()
	test.TestList(validNewNamespace())
}

func TestWatch(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	test := genericregistrytest.New(t, storage.store).ClusterScope()
	test.TestWatch(
		validNewNamespace(),
		// matching labels
		[]labels.Set{},
		// not matching labels
		[]labels.Set{
			{"foo": "bar"},
		},
		// matching fields
		[]fields.Set{
			{"metadata.name": "foo"},
			{"name": "foo"},
		},
		// not matching fields
		[]fields.Set{
			{"metadata.name": "bar"},
		},
	)
}

func TestDeleteNamespaceWithIncompleteFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{corev1.FinalizerOlive},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	obj, immediate, err := storage.Delete(ctx, "foo", rest.ValidateAllObjectFunc, nil)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if immediate {
		t.Fatalf("unexpected immediate flag")
	}
	if ns, ok := obj.(*corev1.Namespace); !ok || obj == nil || ns == nil || ns.Name != namespace.Name {
		t.Fatalf("object not returned by delete")
	}
	// should still exist
	if _, err := storage.Get(ctx, "foo", &metav1.GetOptions{}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateDeletingNamespaceWithIncompleteMetadataFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
			Finalizers:        []string{"example.com/foo"},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	ns, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, err = storage.Update(ctx, "foo", rest.DefaultUpdatedObjectInfo(ns), rest.ValidateAllObjectFunc, rest.ValidateAllObjectUpdateFunc, false, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should still exist
	_, err = storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDeletingNamespaceWithIncompleteSpecFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{corev1.FinalizerOlive},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	if _, _, err = storage.Update(ctx, "foo", rest.DefaultUpdatedObjectInfo(ns), rest.ValidateAllObjectFunc, rest.ValidateAllObjectUpdateFunc, false, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should still exist
	_, err = storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDeletingNamespaceWithCompleteFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
			Finalizers:        []string{"example.com/foo"},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns.(*corev1.Namespace).Finalizers = nil
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	if _, _, err = storage.Update(ctx, "foo", rest.DefaultUpdatedObjectInfo(ns), rest.ValidateAllObjectFunc, rest.ValidateAllObjectUpdateFunc, false, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should not exist
	_, err = storage.Get(ctx, "foo", &metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestFinalizeDeletingNamespaceWithCompleteFinalizers(t *testing.T) {
	// get finalize storage
	etcdStorage, server := registrytest.NewEtcdStorage(t, "")
	restOptions := generic.RESTOptions{StorageConfig: etcdStorage, Decorator: generic.UndecoratedStorage, DeleteCollectionWorkers: 1, ResourcePrefix: "namespaces"}
	storage, _, finalizeStorage, err := NewREST(restOptions)
	if err != nil {
		t.Fatalf("unexpected error from REST storage: %v", err)
	}

	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	defer finalizeStorage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{corev1.FinalizerOlive},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns.(*corev1.Namespace).Spec.Finalizers = nil
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	if _, _, err = finalizeStorage.Update(ctx, "foo", rest.DefaultUpdatedObjectInfo(ns), rest.ValidateAllObjectFunc, rest.ValidateAllObjectUpdateFunc, false, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should not exist
	_, err = storage.Get(ctx, "foo", &metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestFinalizeDeletingNamespaceWithIncompleteMetadataFinalizers(t *testing.T) {
	// get finalize storage
	etcdStorage, server := registrytest.NewEtcdStorage(t, "")
	restOptions := generic.RESTOptions{StorageConfig: etcdStorage, Decorator: generic.UndecoratedStorage, DeleteCollectionWorkers: 1, ResourcePrefix: "namespaces"}
	storage, _, finalizeStorage, err := NewREST(restOptions)
	if err != nil {
		t.Fatalf("unexpected error from REST storage: %v", err)
	}

	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	defer finalizeStorage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
			Finalizers:        []string{"example.com/foo"},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{corev1.FinalizerOlive},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns.(*corev1.Namespace).Spec.Finalizers = nil
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	if _, _, err = finalizeStorage.Update(ctx, "foo", rest.DefaultUpdatedObjectInfo(ns), rest.ValidateAllObjectFunc, rest.ValidateAllObjectUpdateFunc, false, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should still exist
	_, err = storage.Get(ctx, "foo", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteNamespaceWithCompleteFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	key := "namespaces/foo"
	ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
	now := metav1.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo",
			DeletionTimestamp: &now,
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
	if _, _, err := storage.Delete(ctx, "foo", rest.ValidateAllObjectFunc, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// should not exist
	_, err := storage.Get(ctx, "foo", &metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestDeleteWithGCFinalizers(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()

	propagationBackground := metav1.DeletePropagationBackground
	propagationForeground := metav1.DeletePropagationForeground
	propagationOrphan := metav1.DeletePropagationOrphan
	trueVar := true

	var tests = []struct {
		name          string
		deleteOptions *metav1.DeleteOptions

		existingFinalizers  []string
		remainingFinalizers map[string]bool
	}{
		{
			name:          "nil-with-orphan",
			deleteOptions: nil,
			existingFinalizers: []string{
				metav1.FinalizerOrphanDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name:          "nil-with-delete",
			deleteOptions: nil,
			existingFinalizers: []string{
				metav1.FinalizerDeleteDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerDeleteDependents: true,
			},
		},
		{
			name:                "nil-without-finalizers",
			deleteOptions:       nil,
			existingFinalizers:  []string{},
			remainingFinalizers: map[string]bool{},
		},
		{
			name: "propagation-background-with-orphan",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationBackground,
			},
			existingFinalizers: []string{
				metav1.FinalizerOrphanDependents,
			},
			remainingFinalizers: map[string]bool{},
		},
		{
			name: "propagation-background-with-delete",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationBackground,
			},
			existingFinalizers: []string{
				metav1.FinalizerDeleteDependents,
			},
			remainingFinalizers: map[string]bool{},
		},
		{
			name: "propagation-background-without-finalizers",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationBackground,
			},
			existingFinalizers:  []string{},
			remainingFinalizers: map[string]bool{},
		},
		{
			name: "propagation-foreground-with-orphan",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationForeground,
			},
			existingFinalizers: []string{
				metav1.FinalizerOrphanDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerDeleteDependents: true,
			},
		},
		{
			name: "propagation-foreground-with-delete",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationForeground,
			},
			existingFinalizers: []string{
				metav1.FinalizerDeleteDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerDeleteDependents: true,
			},
		},
		{
			name: "propagation-foreground-without-finalizers",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationForeground,
			},
			existingFinalizers: []string{},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerDeleteDependents: true,
			},
		},
		{
			name: "propagation-orphan-with-orphan",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationOrphan,
			},
			existingFinalizers: []string{
				metav1.FinalizerOrphanDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name: "propagation-orphan-with-delete",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationOrphan,
			},
			existingFinalizers: []string{
				metav1.FinalizerDeleteDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name: "propagation-orphan-without-finalizers",
			deleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &propagationOrphan,
			},
			existingFinalizers: []string{},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name: "orphan-dependents-with-orphan",
			deleteOptions: &metav1.DeleteOptions{
				OrphanDependents: &trueVar,
			},
			existingFinalizers: []string{
				metav1.FinalizerOrphanDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name: "orphan-dependents-with-delete",
			deleteOptions: &metav1.DeleteOptions{
				OrphanDependents: &trueVar,
			},
			existingFinalizers: []string{
				metav1.FinalizerDeleteDependents,
			},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
		{
			name: "orphan-dependents-without-finalizers",
			deleteOptions: &metav1.DeleteOptions{
				OrphanDependents: &trueVar,
			},
			existingFinalizers: []string{},
			remainingFinalizers: map[string]bool{
				metav1.FinalizerOrphanDependents: true,
			},
		},
	}

	for _, test := range tests {
		key := "namespaces/" + test.name
		ctx := genericapirequest.WithNamespace(genericapirequest.NewContext(), metav1.NamespaceNone)
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       test.name,
				Finalizers: test.existingFinalizers,
			},
			Spec: corev1.NamespaceSpec{
				Finalizers: []corev1.FinalizerName{},
			},
			Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		}
		if err := storage.store.Storage.Create(ctx, key, namespace, nil, 0, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var obj runtime.Object
		var err error
		ctx = genericapirequest.WithNamespace(genericapirequest.NewContext(), namespace.Name)
		if obj, _, err = storage.Delete(ctx, test.name, rest.ValidateAllObjectFunc, test.deleteOptions); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ns, ok := obj.(*corev1.Namespace)
		if !ok {
			t.Errorf("unexpected object kind: %+v", obj)
		}
		if len(ns.Finalizers) != len(test.remainingFinalizers) {
			t.Errorf("%s: unexpected remaining finalizers: %v", test.name, ns.Finalizers)
		}
		for _, f := range ns.Finalizers {
			if test.remainingFinalizers[f] != true {
				t.Errorf("%s: unexpected finalizer %s", test.name, f)
			}
		}
	}
}

func TestShortNames(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	defer storage.store.DestroyFunc()
	expected := []string{"ns"}
	registrytest.AssertShortNames(t, storage, expected)
}
