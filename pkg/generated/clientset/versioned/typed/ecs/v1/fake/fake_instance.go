/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	ecs_v1 "github.com/houming-wang/ecs-operator/pkg/apis/ecs/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeInstances implements InstanceInterface
type FakeInstances struct {
	Fake *FakeEcsV1
	ns   string
}

var instancesResource = schema.GroupVersionResource{Group: "ecs.easystack.io", Version: "v1", Resource: "instances"}

var instancesKind = schema.GroupVersionKind{Group: "ecs.easystack.io", Version: "v1", Kind: "Instance"}

// Get takes name of the instance, and returns the corresponding instance object, and an error if there is any.
func (c *FakeInstances) Get(name string, options v1.GetOptions) (result *ecs_v1.Instance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(instancesResource, c.ns, name), &ecs_v1.Instance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*ecs_v1.Instance), err
}

// List takes label and field selectors, and returns the list of Instances that match those selectors.
func (c *FakeInstances) List(opts v1.ListOptions) (result *ecs_v1.InstanceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(instancesResource, instancesKind, c.ns, opts), &ecs_v1.InstanceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &ecs_v1.InstanceList{}
	for _, item := range obj.(*ecs_v1.InstanceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested instances.
func (c *FakeInstances) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(instancesResource, c.ns, opts))

}

// Create takes the representation of a instance and creates it.  Returns the server's representation of the instance, and an error, if there is any.
func (c *FakeInstances) Create(instance *ecs_v1.Instance) (result *ecs_v1.Instance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(instancesResource, c.ns, instance), &ecs_v1.Instance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*ecs_v1.Instance), err
}

// Update takes the representation of a instance and updates it. Returns the server's representation of the instance, and an error, if there is any.
func (c *FakeInstances) Update(instance *ecs_v1.Instance) (result *ecs_v1.Instance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(instancesResource, c.ns, instance), &ecs_v1.Instance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*ecs_v1.Instance), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeInstances) UpdateStatus(instance *ecs_v1.Instance) (*ecs_v1.Instance, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(instancesResource, "status", c.ns, instance), &ecs_v1.Instance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*ecs_v1.Instance), err
}

// Delete takes name of the instance and deletes it. Returns an error if one occurs.
func (c *FakeInstances) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(instancesResource, c.ns, name), &ecs_v1.Instance{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeInstances) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(instancesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &ecs_v1.InstanceList{})
	return err
}

// Patch applies the patch and returns the patched instance.
func (c *FakeInstances) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *ecs_v1.Instance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(instancesResource, c.ns, name, data, subresources...), &ecs_v1.Instance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*ecs_v1.Instance), err
}
