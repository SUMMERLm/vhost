/*
Copyright 2017 The Kubernetes Authors.

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

package pkg

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"

	//	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	vhostV1 "github.com/SUMMERLm/vhost/pkg/apis/frontend/v1"
	clientset "github.com/SUMMERLm/vhost/pkg/generated/clientset/versioned"
	vhostscheme "github.com/SUMMERLm/vhost/pkg/generated/clientset/versioned/scheme"
	informers "github.com/SUMMERLm/vhost/pkg/generated/informers/externalversions/frontend/v1"
	listers "github.com/SUMMERLm/vhost/pkg/generated/listers/frontend/v1"
)

const controllerAgentName = "vhost-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a vhost is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a vhost fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by vhost"
	// MessageResourceSynced is the message used for an Event fired when a vhost
	// is synced successfully
	MessageResourceSynced = "vhost synced successfully"
)

// Controller is the controller implementation for vhost resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	dynamicClient dynamic.DynamicClient
	// vhostclientset is a clientset for our own API group
	vhostclientset clientset.Interface
	vhostsLister   listers.VhostLister
	vhostsSynced   cache.InformerSynced
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new vhost controller
func NewController(kubeclientset kubernetes.Interface, dynamicClient dynamic.DynamicClient, vhostclientset clientset.Interface, vhostInformer informers.VhostInformer) *Controller {

	// Create event broadcaster
	utilruntime.Must(vhostscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:  kubeclientset,
		dynamicClient:  dynamicClient,
		vhostclientset: vhostclientset,
		vhostsLister:   vhostInformer.Lister(),
		vhostsSynced:   vhostInformer.Informer().HasSynced,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "vhosts"),
		recorder:       recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when local vhost resources change
	vhostInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addVhost,
		UpdateFunc: controller.updateVhost,
		DeleteFunc: controller.enqueueForDelete,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting vhost controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.vhostsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	// Launch two workers to process vhost resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// vhost resource to be synced.
		if err := c.syncHandler(context.TODO(), key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the vhost resource
// with the current status of the resource.
func (c *Controller) syncHandler(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the vhost resource with this namespace/name
	vhost, err := c.vhostsLister.Vhosts(namespace).Get(name)
	if err != nil {
		// The vhost resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("vhost '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	vhostDel := vhost.DeletionTimestamp.IsZero()
	if !vhostDel {
		err = c.offLine(vhost)
		if err != nil {
			klog.Errorf("Failed to recycle cdn of 'Frontend' %q, error == %v", vhost.Name, err)
			return err
		}
		return nil
	}
	pkgName := vhost.Spec.PkgName
	domainName := vhost.Spec.DomainName
	if pkgName == "" || domainName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: pkgName and domainName must be specified", key))
		return nil
	}
	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	err = c.pkgManage(vhost)
	if err != nil {
		klog.Errorf("Failed to new pkg  of  %q, error == %v", vhost.Name, err)
		return err
	}

	err = c.configManage(vhost)
	if err != nil {
		klog.Errorf("Failed to get cm state of  %q, error == %v", vhost.Name, err)
		return err
	}

	c.recorder.Event(vhost, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		return
	}
}

// addVhost re-queues the Vhost for next scheduled time if there is a
// change in spec.schedule otherwise it re-queues it now
func (c *Controller) addVhost(obj interface{}) {
	vhost := obj.(*vhostV1.Vhost)
	klog.V(4).Infof("adding Vhost %q", vhost)
	c.enqueue(vhost)
}

// updateVhost re-queues the Vhost for next scheduled time if there is a
// change in spec otherwise it re-queues it now
func (c *Controller) updateVhost(old, cur interface{}) {
	oldVhost := old.(*vhostV1.Vhost)
	newVhost := cur.(*vhostV1.Vhost)

	// Decide whether discovery has reported a spec change.
	if reflect.DeepEqual(oldVhost.DeletionTimestamp, newVhost.DeletionTimestamp) && reflect.DeepEqual(oldVhost.Spec, newVhost.Spec) {
		klog.V(4).Infof("no updates on the spec of Vhost %q, skipping syncing", oldVhost.Name)
		return
	}

	klog.V(4).Infof("updating Vhost %q", oldVhost.Name)
	c.enqueue(newVhost)
}

// enqueue takes a Vhost resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Vhost.
func (c *Controller) enqueue(vhost *vhostV1.Vhost) {
	key, err := cache.MetaNamespaceKeyFunc(vhost)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) enqueueForDelete(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	//requeue
	c.workqueue.AddRateLimited(key)
}
