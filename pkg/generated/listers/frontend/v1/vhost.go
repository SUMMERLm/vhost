/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/SUMMERLm/vhost/pkg/apis/frontend/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// VhostLister helps list Vhosts.
// All objects returned here must be treated as read-only.
type VhostLister interface {
	// List lists all Vhosts in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Vhost, err error)
	// Vhosts returns an object that can list and get Vhosts.
	Vhosts(namespace string) VhostNamespaceLister
	VhostListerExpansion
}

// vhostLister implements the VhostLister interface.
type vhostLister struct {
	indexer cache.Indexer
}

// NewVhostLister returns a new VhostLister.
func NewVhostLister(indexer cache.Indexer) VhostLister {
	return &vhostLister{indexer: indexer}
}

// List lists all Vhosts in the indexer.
func (s *vhostLister) List(selector labels.Selector) (ret []*v1.Vhost, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Vhost))
	})
	return ret, err
}

// Vhosts returns an object that can list and get Vhosts.
func (s *vhostLister) Vhosts(namespace string) VhostNamespaceLister {
	return vhostNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// VhostNamespaceLister helps list and get Vhosts.
// All objects returned here must be treated as read-only.
type VhostNamespaceLister interface {
	// List lists all Vhosts in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Vhost, err error)
	// Get retrieves the Vhost from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Vhost, error)
	VhostNamespaceListerExpansion
}

// vhostNamespaceLister implements the VhostNamespaceLister
// interface.
type vhostNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Vhosts in the indexer for a given namespace.
func (s vhostNamespaceLister) List(selector labels.Selector) (ret []*v1.Vhost, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Vhost))
	})
	return ret, err
}

// Get retrieves the Vhost from the indexer for a given namespace and name.
func (s vhostNamespaceLister) Get(name string) (*v1.Vhost, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("vhost"), name)
	}
	return obj.(*v1.Vhost), nil
}
