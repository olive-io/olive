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

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "github.com/olive-io/olive/apis/core/v1"
	corev1 "github.com/olive-io/olive/client/generated/applyconfiguration/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDefinitions implements DefinitionInterface
type FakeDefinitions struct {
	Fake *FakeOliveV1
	ns   string
}

var definitionsResource = v1.SchemeGroupVersion.WithResource("definitions")

var definitionsKind = v1.SchemeGroupVersion.WithKind("Definition")

// Get takes name of the definition, and returns the corresponding definition object, and an error if there is any.
func (c *FakeDefinitions) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Definition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(definitionsResource, c.ns, name), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// List takes label and field selectors, and returns the list of Definitions that match those selectors.
func (c *FakeDefinitions) List(ctx context.Context, opts metav1.ListOptions) (result *v1.DefinitionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(definitionsResource, definitionsKind, c.ns, opts), &v1.DefinitionList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.DefinitionList{ListMeta: obj.(*v1.DefinitionList).ListMeta}
	for _, item := range obj.(*v1.DefinitionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested definitions.
func (c *FakeDefinitions) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(definitionsResource, c.ns, opts))

}

// Create takes the representation of a definition and creates it.  Returns the server's representation of the definition, and an error, if there is any.
func (c *FakeDefinitions) Create(ctx context.Context, definition *v1.Definition, opts metav1.CreateOptions) (result *v1.Definition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(definitionsResource, c.ns, definition), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// Update takes the representation of a definition and updates it. Returns the server's representation of the definition, and an error, if there is any.
func (c *FakeDefinitions) Update(ctx context.Context, definition *v1.Definition, opts metav1.UpdateOptions) (result *v1.Definition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(definitionsResource, c.ns, definition), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeDefinitions) UpdateStatus(ctx context.Context, definition *v1.Definition, opts metav1.UpdateOptions) (*v1.Definition, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(definitionsResource, "status", c.ns, definition), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// Delete takes name of the definition and deletes it. Returns an error if one occurs.
func (c *FakeDefinitions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(definitionsResource, c.ns, name, opts), &v1.Definition{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDefinitions) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(definitionsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.DefinitionList{})
	return err
}

// Patch applies the patch and returns the patched definition.
func (c *FakeDefinitions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Definition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(definitionsResource, c.ns, name, pt, data, subresources...), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied definition.
func (c *FakeDefinitions) Apply(ctx context.Context, definition *corev1.DefinitionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Definition, err error) {
	if definition == nil {
		return nil, fmt.Errorf("definition provided to Apply must not be nil")
	}
	data, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	name := definition.Name
	if name == nil {
		return nil, fmt.Errorf("definition.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(definitionsResource, c.ns, *name, types.ApplyPatchType, data), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeDefinitions) ApplyStatus(ctx context.Context, definition *corev1.DefinitionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Definition, err error) {
	if definition == nil {
		return nil, fmt.Errorf("definition provided to Apply must not be nil")
	}
	data, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	name := definition.Name
	if name == nil {
		return nil, fmt.Errorf("definition.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(definitionsResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1.Definition{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Definition), err
}