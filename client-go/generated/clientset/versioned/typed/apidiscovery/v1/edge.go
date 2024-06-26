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
	apidiscoveryv1 "github.com/olive-io/olive/client-go/generated/applyconfiguration/apidiscovery/v1"
	scheme "github.com/olive-io/olive/client-go/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// EdgesGetter has a method to return a EdgeInterface.
// A group's client should implement this interface.
type EdgesGetter interface {
	Edges(namespace string) EdgeInterface
}

// EdgeInterface has methods to work with Edge resources.
type EdgeInterface interface {
	Create(ctx context.Context, edge *v1.Edge, opts metav1.CreateOptions) (*v1.Edge, error)
	Update(ctx context.Context, edge *v1.Edge, opts metav1.UpdateOptions) (*v1.Edge, error)
	UpdateStatus(ctx context.Context, edge *v1.Edge, opts metav1.UpdateOptions) (*v1.Edge, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Edge, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.EdgeList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Edge, err error)
	Apply(ctx context.Context, edge *apidiscoveryv1.EdgeApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Edge, err error)
	ApplyStatus(ctx context.Context, edge *apidiscoveryv1.EdgeApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Edge, err error)
	EdgeExpansion
}

// edges implements EdgeInterface
type edges struct {
	client rest.Interface
	ns     string
}

// newEdges returns a Edges
func newEdges(c *ApidiscoveryV1Client, namespace string) *edges {
	return &edges{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the edge, and returns the corresponding edge object, and an error if there is any.
func (c *edges) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Edge, err error) {
	result = &v1.Edge{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("edges").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Edges that match those selectors.
func (c *edges) List(ctx context.Context, opts metav1.ListOptions) (result *v1.EdgeList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.EdgeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("edges").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested edges.
func (c *edges) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("edges").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a edge and creates it.  Returns the server's representation of the edge, and an error, if there is any.
func (c *edges) Create(ctx context.Context, edge *v1.Edge, opts metav1.CreateOptions) (result *v1.Edge, err error) {
	result = &v1.Edge{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("edges").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(edge).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a edge and updates it. Returns the server's representation of the edge, and an error, if there is any.
func (c *edges) Update(ctx context.Context, edge *v1.Edge, opts metav1.UpdateOptions) (result *v1.Edge, err error) {
	result = &v1.Edge{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("edges").
		Name(edge.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(edge).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *edges) UpdateStatus(ctx context.Context, edge *v1.Edge, opts metav1.UpdateOptions) (result *v1.Edge, err error) {
	result = &v1.Edge{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("edges").
		Name(edge.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(edge).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the edge and deletes it. Returns an error if one occurs.
func (c *edges) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("edges").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *edges) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("edges").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched edge.
func (c *edges) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Edge, err error) {
	result = &v1.Edge{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("edges").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied edge.
func (c *edges) Apply(ctx context.Context, edge *apidiscoveryv1.EdgeApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Edge, err error) {
	if edge == nil {
		return nil, fmt.Errorf("edge provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(edge)
	if err != nil {
		return nil, err
	}
	name := edge.Name
	if name == nil {
		return nil, fmt.Errorf("edge.Name must be provided to Apply")
	}
	result = &v1.Edge{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("edges").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *edges) ApplyStatus(ctx context.Context, edge *apidiscoveryv1.EdgeApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Edge, err error) {
	if edge == nil {
		return nil, fmt.Errorf("edge provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(edge)
	if err != nil {
		return nil, err
	}

	name := edge.Name
	if name == nil {
		return nil, fmt.Errorf("edge.Name must be provided to Apply")
	}

	result = &v1.Edge{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("edges").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
