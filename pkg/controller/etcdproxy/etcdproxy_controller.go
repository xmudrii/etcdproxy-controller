package etcdproxy

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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
	// EtcdStorageDeployed is used as part of the Event reason when an EtcdStorage resource is successfully synced.
	EtcdStorageDeployed = "EtcdStorageDeployed"

	// EtcdStorageDeployFailure is used as part of the Event reason when an EtcdStorage resource is not synced successfully.
	EtcdStorageDeployFailure = "EtcdStorageDeployFailure"

	// CertificatesDeployFailure is used as part of the Event reason when a Certificates are not generated or deployed successfully.
	CertificatesDeployFailure = "CertificatesDeployFailure"
)

// EtcdProxyController is the controller implementation for EtcdStorage resources
type EtcdProxyController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// etcdProxyClient is a clientset for our own API group
	etcdProxyClient clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced

	servicesLister corev1listers.ServiceLister
	servicesSynced cache.InformerSynced

	etcdstoragesLister listers.EtcdStorageLister
	etcdstoragesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder

	// config is used to wire information used by controller to create Deployments.
	config *EtcdProxyControllerConfig
}

// NewEtcdProxyController returns a new sample controller
func NewEtcdProxyController(
	kubeclientset kubernetes.Interface,
	etcdProxyClient clientset.Interface,
	deploymentsInformer appsinformers.DeploymentInformer,
	servicesInformer corev1informers.ServiceInformer,
	etcdstorageInformer informers.EtcdStorageInformer,
	config *EtcdProxyControllerConfig) *EtcdProxyController {

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
		deploymentsLister:  deploymentsInformer.Lister(),
		deploymentsSynced:  deploymentsInformer.Informer().HasSynced,
		servicesLister:     servicesInformer.Lister(),
		servicesSynced:     servicesInformer.Informer().HasSynced,
		etcdstoragesLister: etcdstorageInformer.Lister(),
		etcdstoragesSynced: etcdstorageInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "EtcdStorages"),
		recorder:           recorder,
		config:             config,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when EtcdStorage resources change
	etcdstorageInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueEtcdStorage,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueEtcdStorage(new)
		},
	})

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a EtcdStorage resource will enqueue that EtcdStorage resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newRSet := new.(*appsv1.Deployment)
			oldRSet := old.(*appsv1.Deployment)
			if newRSet.ResourceVersion == oldRSet.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployments will always have different RVs.
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
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.servicesSynced, c.etcdstoragesSynced); !ok {
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

	// This prevents syncHandler to continue in case an EtcdStorage resource is being deleted.
	// Otherwise, the controller ends up in the Deployment recreation loop until GC doesn't
	// delete the EtcdStorage resource.
	if !etcdstorage.DeletionTimestamp.IsZero() {
		glog.V(2).Infof("EtcdStorage %s is being terminated.", etcdstorage.Name)
		return nil
	}

	etcdstorageCondition := etcdstoragev1alpha1.EtcdStorageCondition{
		Type:   etcdstoragev1alpha1.Deployed,
		Status: etcdstoragev1alpha1.ConditionUnknown,
	}

	var errs []error
	var certErrs []error
	// Deploy Server Etcd Proxy certificates.
	if err = c.ensureClientCertificates(etcdstorage); err != nil {
		certErrs = append(certErrs, err)
	}
	if err = c.ensureServerCertificates(etcdstorage); err != nil {
		certErrs = append(certErrs, err)
	}

	// Etcd proxy Deployment.
	deployment, err := c.deploymentsLister.Deployments(c.config.ControllerNamespace).Get(deploymentName(etcdstorage))
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(c.config.ControllerNamespace).Create(newDeployment(
			etcdstorage, c.config.ControllerNamespace, etcdstorage.Name,
			c.config.ProxyImage, c.config.CoreEtcd.CAConfigMapName, c.config.CoreEtcd.CertSecretName,
			c.config.CoreEtcd.URLs))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		etcdstorageCondition.Status = etcdstoragev1alpha1.ConditionFalse
		etcdstorageCondition.Reason = "FailedDeploying"
		etcdstorageCondition.Message = err.Error()
		errs = append(errs, err)
	}

	// If the ReplicaSet is not controlled by this EtcdStorage resource, we should try to update Owner reference.
	if !metav1.IsControlledBy(deployment, etcdstorage) {
		deployment.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
		})

		deployment, err = c.kubeclientset.AppsV1().Deployments(c.config.ControllerNamespace).Update(deployment)
		if err != nil {
			glog.V(2).Infof("Unable to update OwnerRef for ReplicaSet %s: %v", deployment.Name, err)
		}
	}

	// Create Service to expose the etcdproxy pod.
	serviceName := fmt.Sprintf("etcd-%s", etcdstorage.ObjectMeta.Name)
	service, err := c.servicesLister.Services(c.config.ControllerNamespace).Get(serviceName)
	if errors.IsNotFound(err) {
		service, err = c.kubeclientset.CoreV1().Services(c.config.ControllerNamespace).Create(newService(
			etcdstorage, c.config.ControllerNamespace))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		etcdstorageCondition.Status = etcdstoragev1alpha1.ConditionFalse
		etcdstorageCondition.Reason = "FailedDeploying"
		etcdstorageCondition.Message = err.Error()
		errs = append(errs, err)
	}

	// If the Service is not controlled by this EtcdStorage resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(service, etcdstorage) {
		service.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(etcdstorage, etcdstoragev1alpha1.SchemeGroupVersion.WithKind("EtcdStorage")),
		})

		service, err = c.kubeclientset.CoreV1().Services(c.config.ControllerNamespace).Update(service)
		if err != nil {
			glog.V(2).Infof("Unable to update OwnerRef for Secret %s: %v", service.Name, err)
		}
	}

	// Finally, we update the status block of the EtcdStorage resource to reflect the
	// current state of the world
	if etcdstorageCondition.Status == etcdstoragev1alpha1.ConditionUnknown {
		etcdstorageCondition = etcdstoragev1alpha1.EtcdStorageCondition{
			Type:    etcdstoragev1alpha1.Deployed,
			Status:  etcdstoragev1alpha1.ConditionTrue,
			Reason:  "Deployed",
			Message: "etcdproxy replicaset and service created",
		}
	}

	_, err = c.updateEtcdStorageStatus(etcdstorage, etcdstorageCondition)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		c.recorder.Event(etcdstorage, corev1.EventTypeNormal,
			EtcdStorageDeployed, fmt.Sprintf("EtcdStorage %s synced successfully", etcdstorage.Name))
	}
	if len(certErrs) != 0 {
		errs = append(errs, certErrs...)
		c.recorder.Event(etcdstorage, corev1.EventTypeWarning,
			CertificatesDeployFailure, fmt.Sprintf("Unable to deploy EtcdProxy Certificates for EtcdStorage %s: %v",
				etcdstorage.Name, utilerrors.NewAggregate(errs)))
	}

	return utilerrors.NewAggregate(errs)
}

func (c *EtcdProxyController) updateEtcdStorageStatus(etcdstorage *etcdstoragev1alpha1.EtcdStorage,
	condition etcdstoragev1alpha1.EtcdStorageCondition) (*etcdstoragev1alpha1.EtcdStorage, error) {
	etcdstorageCopy := etcdstorage.DeepCopy()
	etcdstoragev1alpha1.SetEtcdStorageCondition(etcdstorageCopy, condition)

	// We're not updating the EtcdStorage resource if there are no Status changes between new and old objects
	// in order to prevent Update loops.
	if equality.Semantic.DeepEqual(etcdstorageCopy.Status, etcdstorage.Status) {
		return etcdstorage, nil
	}

	etcdstorageCopy, err := c.etcdProxyClient.EtcdV1alpha1().EtcdStorages().UpdateStatus(etcdstorageCopy)
	return etcdstorageCopy, err
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
