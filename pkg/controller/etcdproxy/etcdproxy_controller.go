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

package etcdproxy

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	etcdstoragev1alpha1 "github.com/xmudrii/etcdproxy-controller/pkg/apis/etcd/v1alpha1"
	clientset "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned"
	samplescheme "github.com/xmudrii/etcdproxy-controller/pkg/client/clientset/versioned/scheme"
	informers "github.com/xmudrii/etcdproxy-controller/pkg/client/informers/externalversions/etcd/v1alpha1"
	listers "github.com/xmudrii/etcdproxy-controller/pkg/client/listers/etcd/v1alpha1"
)

const httpUserAgentName = "etcdproxy-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a EtcdStorage is synced
	SuccessSynced = "Synced"
	// ResourceReclaimed is used as part of the Event 'reason' when a replicaset or service already exists
	// and EtcdStorage reclaims it.
	ResourceReclaimed = "ResourceReclaimed"

	// ErrResourceReclaimed is used as port of the Event 'reason' when reclaiming a resource fails.
	ErrResourceReclaimed = "ErrResourceReclaimed"
	// ErrUnknown is used as part of the Event 'reason' when a EtcdStorage fails
	// to get, create, or update resource.
	ErrUnknown = "ErrUnknown"

	// ResourceReclaimedReason is the message used for Events when a resource
	// fails to sync due to a ReplicaSet already existing
	ResourceReclaimedReason = "Resource %q already exists and is set to be managed by EtcdStorage"
	// MessageResourceSynced is the message used for an Event fired when a EtcdStorage
	// is synced successfully
	MessageResourceSynced = "EtcdStorage synced successfully"
	// MessageErrResourceReclaimed is the message used for an Event fired when a ErrResourceReclaimed occurs.
	MessageErrResourceReclaimed = "Unable to put EtcdStorage OwnerReference to resource %q"
)

// CoreEtcdOptions type is used to wire the core etcd information used by controller to create ReplicaSets.
type CoreEtcdOptions struct {
	URL             string
	CAConfigMapName string
	CertSecretName  string
}

// EtcdProxyOptions type is used to pass information from cli to the controller.
type EtcdProxyOptions struct {
	CoreEtcd CoreEtcdOptions

	// ControllerNamespace is name of namespace where controller is deployed.
	ControllerNamespace string

	// ProxyImage is name of the etcd image to be used for etcd-proxy ReplicaSet creation.
	ProxyImage string
}

// EtcdProxyController is the controller implementation for EtcdStorage resources
type EtcdProxyController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// etcdProxyClient is a clientset for our own API group
	etcdProxyClient clientset.Interface

	replicasetsLister appslisters.ReplicaSetLister
	replicasetsSynced cache.InformerSynced

	servicesLister corev1listers.ServiceLister
	servicesSynced cache.InformerSynced

	etcdstoragesLister listers.EtcdStorageLister
	etcdstoragesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder

	// etcdProxyOptions is used to wire information used by controller to create ReplicaSets.
	etcdProxyOptions *EtcdProxyOptions
}

// NewEtcdProxyController returns a new sample controller
func NewEtcdProxyController(
	kubeclientset kubernetes.Interface,
	etcdProxyClient clientset.Interface,
	replicasetsInformer appsinformers.ReplicaSetInformer,
	servicesInformer corev1informers.ServiceInformer,
	etcdstorageInformer informers.EtcdStorageInformer,
	etcdProxyOptions *EtcdProxyOptions) *EtcdProxyController {

	// Create event broadcaster
	// Add the controller types to the default Kubernetes Scheme so Events can be logged for the controller types.
	samplescheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: httpUserAgentName})

	controller := &EtcdProxyController{
		kubeclientset:      kubeclientset,
		etcdProxyClient:    etcdProxyClient,
		replicasetsLister:  replicasetsInformer.Lister(),
		replicasetsSynced:  replicasetsInformer.Informer().HasSynced,
		servicesLister:     servicesInformer.Lister(),
		servicesSynced:     servicesInformer.Informer().HasSynced,
		etcdstoragesLister: etcdstorageInformer.Lister(),
		etcdstoragesSynced: etcdstorageInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "EtcdStorages"),
		recorder:           recorder,
		etcdProxyOptions:   etcdProxyOptions,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when EtcdStorage resources change
	etcdstorageInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueEtcdStorage,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueEtcdStorage(new)
		},
	})

	// Set up an event handler for when ReplicaSet resources change. This
	// handler will lookup the owner of the given ReplicaSet, and if it is
	// owned by a EtcdStorage resource will enqueue that EtcdStorage resource for
	// processing. This way, we don't need to implement custom logic for
	// handling ReplicaSet resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	replicasetsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newRSet := new.(*appsv1.ReplicaSet)
			oldRSet := old.(*appsv1.ReplicaSet)
			if newRSet.ResourceVersion == oldRSet.ResourceVersion {
				// Periodic resync will send update events for all known ReplicaSets.
				// Two different versions of the same ReplicaSet will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	servicesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newSvc := new.(*corev1.Service)
			oldSvc := old.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				// Periodic resync will send update events for all known Services.
				// Two different versions of the same Service will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *EtcdProxyController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting EtcdStorage controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.replicasetsSynced, c.servicesSynced, c.etcdstoragesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process EtcdStorage resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *EtcdProxyController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *EtcdProxyController) processNextWorkItem() bool {
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
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// EtcdStorage resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the EtcdStorage resource
// with the current status of the resource.
func (c *EtcdProxyController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the EtcdStorage resource with this namespace/name
	etcdstorage, err := c.etcdstoragesLister.Get(name)
	if err != nil {
		// The EtcdStorage resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("etcdstorage '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	replicaset, err := c.replicasetsLister.ReplicaSets(c.etcdProxyOptions.ControllerNamespace).Get(replicaSetName(etcdstorage))
	if errors.IsNotFound(err) {
		replicaset, err = c.kubeclientset.AppsV1().ReplicaSets(c.etcdProxyOptions.ControllerNamespace).Create(newReplicaSet(
			etcdstorage, c.etcdProxyOptions.ControllerNamespace, etcdstorage.Name,
			c.etcdProxyOptions.ProxyImage, c.etcdProxyOptions.CoreEtcd.URL,
			c.etcdProxyOptions.CoreEtcd.CAConfigMapName, c.etcdProxyOptions.CoreEtcd.CertSecretName))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}

	// If the ReplicaSet is not controlled by this EtcdStorage resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(replicaset, etcdstorage) {
		replicaset.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
		})
		replicaset, err = c.kubeclientset.AppsV1().ReplicaSets(c.etcdProxyOptions.ControllerNamespace).Update(replicaset)
		if err != nil {
			msg := fmt.Sprintf(MessageErrResourceReclaimed, replicaset.Name)
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrResourceReclaimed, msg)
			return err
		}
		msg := fmt.Sprintf(ResourceReclaimedReason, replicaset.Name)
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ResourceReclaimed, msg)
	}

	// Create Service to expose the etcdproxy pod.
	serviceName := fmt.Sprintf("etcd-%s", etcdstorage.ObjectMeta.Name)
	service, err := c.servicesLister.Services(c.etcdProxyOptions.ControllerNamespace).Get(serviceName)
	if errors.IsNotFound(err) {
		service, err = c.kubeclientset.CoreV1().Services(c.etcdProxyOptions.ControllerNamespace).Create(newService(
			etcdstorage, c.etcdProxyOptions.ControllerNamespace))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrUnknown, err.Error())
		return err
	}

	// If the Service is not controlled by this EtcdStorage resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(service, etcdstorage) {
		service.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
		})
		service, err = c.kubeclientset.CoreV1().Services(c.etcdProxyOptions.ControllerNamespace).Update(service)
		if err != nil {
			msg := fmt.Sprintf(MessageErrResourceReclaimed, service.Name)
			c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ErrResourceReclaimed, msg)
			return err
		}
		msg := fmt.Sprintf(ResourceReclaimedReason, service.Name)
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning, ResourceReclaimed, msg)
	}

	// TODO(xmudrii): Add CR status updating once Status subresource is implemented.

	c.recorder.Event(etcdstorage, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueEtcdStorage takes a EtcdStorage resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than EtcdStorage.
func (c *EtcdProxyController) enqueueEtcdStorage(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the EtcdStorage resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that EtcdStorage resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *EtcdProxyController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a EtcdStorage, we should not do anything more
		// with it.
		if ownerRef.Kind != "EtcdStorage" {
			return
		}

		etcdstorage, err := c.etcdstoragesLister.Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of etcdstorage '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueEtcdStorage(etcdstorage)
		return
	}
}
