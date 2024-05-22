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

	v1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	apidiscoveryv1 "github.com/olive-io/olive/client/generated/applyconfiguration/apidiscovery/v1"
	scheme "github.com/olive-io/olive/client/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// YardsGetter has a method to return a YardInterface.
// A group's client should implement this interface.
type YardsGetter interface {
	Yards(namespace string) YardInterface
}

// YardInterface has methods to work with Yard resources.
type YardInterface interface {
	Create(ctx context.Context, yard *v1.Yard, opts metav1.CreateOptions) (*v1.Yard, error)
	Update(ctx context.Context, yard *v1.Yard, opts metav1.UpdateOptions) (*v1.Yard, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Yard, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.YardList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Yard, err error)
	Apply(ctx context.Context, yard *apidiscoveryv1.YardApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Yard, err error)
	YardExpansion
}

// yards implements YardInterface
type yards struct {
	client rest.Interface
	ns     string
}

// newYards returns a Yards
func newYards(c *DiscoveryV1Client, namespace string) *yards {
	return &yards{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the yard, and returns the corresponding yard object, and an error if there is any.
func (c *yards) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Yard, err error) {
	result = &v1.Yard{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("yards").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Yards that match those selectors.
func (c *yards) List(ctx context.Context, opts metav1.ListOptions) (result *v1.YardList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.YardList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("yards").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested yards.
func (c *yards) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("yards").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a yard and creates it.  Returns the server's representation of the yard, and an error, if there is any.
func (c *yards) Create(ctx context.Context, yard *v1.Yard, opts metav1.CreateOptions) (result *v1.Yard, err error) {
	result = &v1.Yard{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("yards").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(yard).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a yard and updates it. Returns the server's representation of the yard, and an error, if there is any.
func (c *yards) Update(ctx context.Context, yard *v1.Yard, opts metav1.UpdateOptions) (result *v1.Yard, err error) {
	result = &v1.Yard{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("yards").
		Name(yard.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(yard).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the yard and deletes it. Returns an error if one occurs.
func (c *yards) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("yards").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *yards) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("yards").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched yard.
func (c *yards) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Yard, err error) {
	result = &v1.Yard{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("yards").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied yard.
func (c *yards) Apply(ctx context.Context, yard *apidiscoveryv1.YardApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Yard, err error) {
	if yard == nil {
		return nil, fmt.Errorf("yard provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(yard)
	if err != nil {
		return nil, err
	}
	name := yard.Name
	if name == nil {
		return nil, fmt.Errorf("yard.Name must be provided to Apply")
	}
	result = &v1.Yard{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("yards").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
