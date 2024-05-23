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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1 "github.com/olive-io/olive/apis/core/v1"
	corev1 "github.com/olive-io/olive/client/generated/applyconfiguration/core/v1"
	scheme "github.com/olive-io/olive/client/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ProcessInstancesGetter has a method to return a ProcessInstanceInterface.
// A group's client should implement this interface.
type ProcessInstancesGetter interface {
	ProcessInstances(namespace string) ProcessInstanceInterface
}

// ProcessInstanceInterface has methods to work with ProcessInstance resources.
type ProcessInstanceInterface interface {
	Create(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.CreateOptions) (*v1.ProcessInstance, error)
	Update(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.UpdateOptions) (*v1.ProcessInstance, error)
	UpdateStatus(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.UpdateOptions) (*v1.ProcessInstance, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ProcessInstance, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ProcessInstanceList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ProcessInstance, err error)
	Apply(ctx context.Context, processInstance *corev1.ProcessInstanceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ProcessInstance, err error)
	ApplyStatus(ctx context.Context, processInstance *corev1.ProcessInstanceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ProcessInstance, err error)
	ProcessInstanceExpansion
}

// processInstances implements ProcessInstanceInterface
type processInstances struct {
	client rest.Interface
	ns     string
}

// newProcessInstances returns a ProcessInstances
func newProcessInstances(c *OliveV1Client, namespace string) *processInstances {
	return &processInstances{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the processInstance, and returns the corresponding processInstance object, and an error if there is any.
func (c *processInstances) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ProcessInstance, err error) {
	result = &v1.ProcessInstance{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("processinstances").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ProcessInstances that match those selectors.
func (c *processInstances) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ProcessInstanceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ProcessInstanceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("processinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested processInstances.
func (c *processInstances) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("processinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a processInstance and creates it.  Returns the server's representation of the processInstance, and an error, if there is any.
func (c *processInstances) Create(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.CreateOptions) (result *v1.ProcessInstance, err error) {
	result = &v1.ProcessInstance{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("processinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(processInstance).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a processInstance and updates it. Returns the server's representation of the processInstance, and an error, if there is any.
func (c *processInstances) Update(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.UpdateOptions) (result *v1.ProcessInstance, err error) {
	result = &v1.ProcessInstance{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("processinstances").
		Name(processInstance.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(processInstance).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *processInstances) UpdateStatus(ctx context.Context, processInstance *v1.ProcessInstance, opts metav1.UpdateOptions) (result *v1.ProcessInstance, err error) {
	result = &v1.ProcessInstance{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("processinstances").
		Name(processInstance.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(processInstance).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the processInstance and deletes it. Returns an error if one occurs.
func (c *processInstances) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("processinstances").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *processInstances) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("processinstances").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched processInstance.
func (c *processInstances) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ProcessInstance, err error) {
	result = &v1.ProcessInstance{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("processinstances").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied processInstance.
func (c *processInstances) Apply(ctx context.Context, processInstance *corev1.ProcessInstanceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ProcessInstance, err error) {
	if processInstance == nil {
		return nil, fmt.Errorf("processInstance provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(processInstance)
	if err != nil {
		return nil, err
	}
	name := processInstance.Name
	if name == nil {
		return nil, fmt.Errorf("processInstance.Name must be provided to Apply")
	}
	result = &v1.ProcessInstance{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("processinstances").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *processInstances) ApplyStatus(ctx context.Context, processInstance *corev1.ProcessInstanceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ProcessInstance, err error) {
	if processInstance == nil {
		return nil, fmt.Errorf("processInstance provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(processInstance)
	if err != nil {
		return nil, err
	}

	name := processInstance.Name
	if name == nil {
		return nil, fmt.Errorf("processInstance.Name must be provided to Apply")
	}

	result = &v1.ProcessInstance{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("processinstances").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
