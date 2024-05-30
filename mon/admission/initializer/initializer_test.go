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

package initializer_test

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// TestWantsAuthorizer ensures that the authorizer is injected
// when the WantsAuthorizer interface is implemented by a plugin.
func TestWantsAuthorizer(t *testing.T) {
	target := initializer.New(nil, nil, nil, &TestAuthorizer{}, nil, nil, nil)
	wantAuthorizerAdmission := &WantAuthorizerAdmission{}
	target.Initialize(wantAuthorizerAdmission)
	if wantAuthorizerAdmission.auth == nil {
		t.Errorf("expected authorizer to be initialized but found nil")
	}
}

// TestWantsExternalKubeClientSet ensures that the clientset is injected
// when the WantsExternalKubeClientSet interface is implemented by a plugin.
func TestWantsExternalKubeClientSet(t *testing.T) {
	cs := &fake.Clientset{}
	target := initializer.New(cs, nil, nil, &TestAuthorizer{}, nil, nil, nil)
	wantExternalKubeClientSet := &WantExternalKubeClientSet{}
	target.Initialize(wantExternalKubeClientSet)
	if wantExternalKubeClientSet.cs != cs {
		t.Errorf("expected clientset to be initialized")
	}
}

// TestWantsExternalKubeInformerFactory ensures that the informer factory is injected
// when the WantsExternalKubeInformerFactory interface is implemented by a plugin.
func TestWantsExternalKubeInformerFactory(t *testing.T) {
	cs := &fake.Clientset{}
	sf := informers.NewSharedInformerFactory(cs, time.Duration(1)*time.Second)
	target := initializer.New(cs, nil, sf, &TestAuthorizer{}, nil, nil, nil)
	wantExternalKubeInformerFactory := &WantExternalKubeInformerFactory{}
	target.Initialize(wantExternalKubeInformerFactory)
	if wantExternalKubeInformerFactory.sf != sf {
		t.Errorf("expected informer factory to be initialized")
	}
}

// TestWantsShutdownSignal ensures that the shutdown signal is injected
// when the WantsShutdownSignal interface is implemented by a plugin.
func TestWantsShutdownNotification(t *testing.T) {
	stopCh := make(chan struct{})
	target := initializer.New(nil, nil, nil, &TestAuthorizer{}, nil, stopCh, nil)
	wantDrainedNotification := &WantDrainedNotification{}
	target.Initialize(wantDrainedNotification)
	if wantDrainedNotification.stopCh == nil {
		t.Errorf("expected stopCh to be initialized but found nil")
	}
}

// WantExternalKubeInformerFactory is a test stub that fulfills the WantsExternalKubeInformerFactory interface
type WantExternalKubeInformerFactory struct {
	sf informers.SharedInformerFactory
}

func (self *WantExternalKubeInformerFactory) SetExternalKubeInformerFactory(sf informers.SharedInformerFactory) {
	self.sf = sf
}
func (self *WantExternalKubeInformerFactory) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (self *WantExternalKubeInformerFactory) Handles(o admission.Operation) bool { return false }
func (self *WantExternalKubeInformerFactory) ValidateInitialization() error      { return nil }

var _ admission.Interface = &WantExternalKubeInformerFactory{}
var _ initializer.WantsExternalKubeInformerFactory = &WantExternalKubeInformerFactory{}

// WantExternalKubeClientSet is a test stub that fulfills the WantsExternalKubeClientSet interface
type WantExternalKubeClientSet struct {
	cs kubernetes.Interface
}

func (self *WantExternalKubeClientSet) SetExternalKubeClientSet(cs kubernetes.Interface) {
	self.cs = cs
}
func (self *WantExternalKubeClientSet) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (self *WantExternalKubeClientSet) Handles(o admission.Operation) bool { return false }
func (self *WantExternalKubeClientSet) ValidateInitialization() error      { return nil }

var _ admission.Interface = &WantExternalKubeClientSet{}
var _ initializer.WantsExternalKubeClientSet = &WantExternalKubeClientSet{}

// WantAuthorizerAdmission is a test stub that fulfills the WantsAuthorizer interface.
type WantAuthorizerAdmission struct {
	auth authorizer.Authorizer
}

func (self *WantAuthorizerAdmission) SetAuthorizer(a authorizer.Authorizer) { self.auth = a }
func (self *WantAuthorizerAdmission) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (self *WantAuthorizerAdmission) Handles(o admission.Operation) bool { return false }
func (self *WantAuthorizerAdmission) ValidateInitialization() error      { return nil }

var _ admission.Interface = &WantAuthorizerAdmission{}
var _ initializer.WantsAuthorizer = &WantAuthorizerAdmission{}

// WantDrainedNotification is a test stub that filfills the WantsDrainedNotification interface.
type WantDrainedNotification struct {
	stopCh <-chan struct{}
}

func (self *WantDrainedNotification) SetDrainedNotification(stopCh <-chan struct{}) {
	self.stopCh = stopCh
}
func (self *WantDrainedNotification) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (self *WantDrainedNotification) Handles(o admission.Operation) bool { return false }
func (self *WantDrainedNotification) ValidateInitialization() error      { return nil }

var _ admission.Interface = &WantDrainedNotification{}
var _ initializer.WantsDrainedNotification = &WantDrainedNotification{}

// TestAuthorizer is a test stub that fulfills the WantsAuthorizer interface.
type TestAuthorizer struct{}

func (t *TestAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	return authorizer.DecisionNoOpinion, "", nil
}

func TestRESTMapperAdmissionPlugin(t *testing.T) {
	initializer := initializer.New(nil, nil, nil, &TestAuthorizer{}, nil, nil, &doNothingRESTMapper{})
	wantsRESTMapperAdmission := &WantsRESTMapperAdmissionPlugin{}
	initializer.Initialize(wantsRESTMapperAdmission)

	if wantsRESTMapperAdmission.mapper == nil {
		t.Errorf("Expected REST mapper to be initialized but found nil")
	}
}

type WantsRESTMapperAdmissionPlugin struct {
	doNothingAdmission
	doNothingPluginInitialization
	mapper meta.RESTMapper
}

func (p *WantsRESTMapperAdmissionPlugin) SetRESTMapper(mapper meta.RESTMapper) {
	p.mapper = mapper
}

type doNothingRESTMapper struct{}

func (doNothingRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
func (doNothingRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}
func (doNothingRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}
func (doNothingRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, nil
}
func (doNothingRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	return nil, nil
}
func (doNothingRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	return nil, nil
}
func (doNothingRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return "", nil
}

type doNothingAdmission struct{}

func (doNothingAdmission) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (doNothingAdmission) Handles(o admission.Operation) bool { return false }
func (doNothingAdmission) Validate() error                    { return nil }

type doNothingPluginInitialization struct{}

func (doNothingPluginInitialization) ValidateInitialization() error { return nil }
