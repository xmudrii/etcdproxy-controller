/*
Copyright 2018 The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEtcdStorages implements EtcdStorageInterface
type FakeEtcdStorages struct {
	Fake *FakeEtcdV1alpha1
}

var etcdstoragesResource = schema.GroupVersionResource{Group: "etcd.xmudrii.com", Version: "v1alpha1", Resource: "etcdstorages"}

var etcdstoragesKind = schema.GroupVersionKind{Group: "etcd.xmudrii.com", Version: "v1alpha1", Kind: "EtcdStorage"}

// Get takes name of the etcdStorage, and returns the corresponding etcdStorage object, and an error if there is any.
func (c *FakeEtcdStorages) Get(name string, options v1.GetOptions) (result *v1alpha1.EtcdStorage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(etcdstoragesResource, name), &v1alpha1.EtcdStorage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EtcdStorage), err
}

// List takes label and field selectors, and returns the list of EtcdStorages that match those selectors.
func (c *FakeEtcdStorages) List(opts v1.ListOptions) (result *v1alpha1.EtcdStorageList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(etcdstoragesResource, etcdstoragesKind, opts), &v1alpha1.EtcdStorageList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.EtcdStorageList{}
	for _, item := range obj.(*v1alpha1.EtcdStorageList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested etcdStorages.
func (c *FakeEtcdStorages) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(etcdstoragesResource, opts))
}

// Create takes the representation of a etcdStorage and creates it.  Returns the server's representation of the etcdStorage, and an error, if there is any.
func (c *FakeEtcdStorages) Create(etcdStorage *v1alpha1.EtcdStorage) (result *v1alpha1.EtcdStorage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(etcdstoragesResource, etcdStorage), &v1alpha1.EtcdStorage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EtcdStorage), err
}

// Update takes the representation of a etcdStorage and updates it. Returns the server's representation of the etcdStorage, and an error, if there is any.
func (c *FakeEtcdStorages) Update(etcdStorage *v1alpha1.EtcdStorage) (result *v1alpha1.EtcdStorage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(etcdstoragesResource, etcdStorage), &v1alpha1.EtcdStorage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EtcdStorage), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeEtcdStorages) UpdateStatus(etcdStorage *v1alpha1.EtcdStorage) (*v1alpha1.EtcdStorage, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(etcdstoragesResource, "status", etcdStorage), &v1alpha1.EtcdStorage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EtcdStorage), err
}

// Delete takes name of the etcdStorage and deletes it. Returns an error if one occurs.
func (c *FakeEtcdStorages) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(etcdstoragesResource, name), &v1alpha1.EtcdStorage{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEtcdStorages) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(etcdstoragesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.EtcdStorageList{})
	return err
}

// Patch applies the patch and returns the patched etcdStorage.
func (c *FakeEtcdStorages) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.EtcdStorage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(etcdstoragesResource, name, data, subresources...), &v1alpha1.EtcdStorage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EtcdStorage), err
}
