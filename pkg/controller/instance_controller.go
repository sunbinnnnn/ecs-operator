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

package controller

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	//appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	ecsv1 "github.com/houming-wang/ecs-operator/pkg/apis/ecs/v1"
	clientset "github.com/houming-wang/ecs-operator/pkg/generated/clientset/versioned"
	ecsscheme "github.com/houming-wang/ecs-operator/pkg/generated/clientset/versioned/scheme"
	informers "github.com/houming-wang/ecs-operator/pkg/generated/informers/externalversions"
	listers "github.com/houming-wang/ecs-operator/pkg/generated/listers/ecs/v1"
	"github.com/houming-wang/ecs-operator/pkg/utils/openstack"
)

const (
	controllerAgentName = "instance-controller"

	// SuccessSynced is used as part of the Event 'reason' when a Instance is synced
	SuccessSynced = "Synced"

	// MessageResourceSynced is the message used for an Event fired when a VirtualMacine
	// is synced successfully
	MessageResourceSynced = "Instance synced successfully"

	// maxRetries is the number of times a deployment will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a deployment is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

type cachedInstance struct {
	// The cached state of the instance
	inst *ecsv1.Instance
}

type instanceCache struct {
	mu         sync.Mutex // protects instanceMap
	// this instanceMap is a local cache for instance.
	// mainly used for instance delete event, because instance crd object is immediately from apiserver/etcd,
	// but bounded OpenStack Instance is not yet be deleted. We use this cache to maintain OpenStack instance info.
	instanceMap map[string]*cachedInstance
}

// Controller is the controller implementation for Foo resources
type InstanceController struct {
	// ecsclientset is a clientset for our own API group
	ecsclientset clientset.Interface

	instancesLister listers.InstanceLister
	instancesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// OpenStack Service Client
	osService *openstack.Service

	// local instance cache
	localCache *instanceCache
}

// NewInstanceController returns a new instance controller
func NewInstanceController(
	kubeclientset kubernetes.Interface,
	ecsclientset clientset.Interface,
	ecsInformerFactory informers.SharedInformerFactory,
	osService *openstack.Service) *InstanceController {

	instanceInformer := ecsInformerFactory.Ecs().V1().Instances()

	// Create event broadcaster
	// Add ecs types to the default Kubernetes Scheme so Events can be
	// logged for ecs  types.
	glog.V(4).Info("Creating event broadcaster")
	ecsscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	ic := &InstanceController{
		ecsclientset:    ecsclientset,
		instancesLister: instanceInformer.Lister(),
		instancesSynced: instanceInformer.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Instances"),
		recorder:        recorder,
		osService:       osService,
		localCache:      &instanceCache{instanceMap: make(map[string]*cachedInstance)},
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when CRUD to Instance resources
	instanceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ic.addInstance,
		UpdateFunc: ic.updateInstance,
		DeleteFunc: ic.deleteInstance,
	})

	return ic
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (ic *InstanceController) Run(workers int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer ic.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Instance controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, ic.instancesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Instance resources
	for i := 0; i < workers; i++ {
		go wait.Until(ic.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (ic *InstanceController) runWorker() {
	for ic.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (ic *InstanceController) processNextWorkItem() bool {
	key, quit := ic.workqueue.Get()
	if quit {
		return false
	}
	defer ic.workqueue.Done(key)

	err := ic.syncHandler(key.(string))
	ic.handleErr(err, key)

	return true
}

func (ic *InstanceController) handleErr(err error, key interface{}) {
	if err == nil {
		ic.workqueue.Forget(key)
		return
	}

	if ic.workqueue.NumRequeues(key) < maxRetries {
		glog.V(2).Infof("Error syncing instance %v: %v", key, err)
		ic.workqueue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	glog.V(2).Infof("Dropping instance %q out of the queue: %v", key, err)
	ic.workqueue.Forget(key)
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Instance resource
// with the current status of the resource.
func (ic *InstanceController) syncHandler(key string) error {
	startTime := time.Now()
	glog.V(4).Infof("Started syncing instance %q (%v)", key, startTime)
	defer func() {
		glog.V(4).Infof("Finished syncing instance %q (%v)", key, time.Since(startTime))
	}()

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Instance resource with this namespace/name
	instance, err := ic.instancesLister.Instances(namespace).Get(name)
	if err != nil {
		// The Instance resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			return ic.processInstanceDelete(key)
		}
		return err
	}

	// Instance business logic to OpenStack
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		instanceCopy := instance.DeepCopy()
		if instance.Status.ID != "" {
			osInstance, err := ic.osService.InstanceGet(instance.Status.ID)
			if err != nil {
				if strings.Contains(err.Error(), "Resource not found") {
					instanceCopy.Status.ID = ""
					instanceCopy.Status.Status = ""
					instanceCopy.Status.TenantID = ""
				}
			} else {
				if osInstance.Status == "ERROR" && time.Since(instance.ObjectMeta.CreationTimestamp.Time) > time.Minute * 5 {
					return ic.instanceDelete(instance)
				}
				instanceCopy.Status.Status = osInstance.Status
				instanceCopy.Status.TenantID = osInstance.TenantID
			}
		} else {
			id, err := ic.instanceCreate(instance)
			if err != nil {
				return err
			}
			instanceCopy.Status.ID = id
		}

		_, err := ic.ecsclientset.EcsV1().Instances(instance.Namespace).Update(instanceCopy)
		if err != nil {
			return err
		}

		_, ok := ic.localCache.get(key)
		if !ok {
			ic.localCache.getOrCreate(key)
			ic.localCache.set(key, &cachedInstance{inst: instanceCopy})
		}
	}

	ic.recorder.Event(instance, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (ic *InstanceController) addInstance(obj interface{}) {
	i := obj.(*ecsv1.Instance)
	glog.V(4).Infof("Adding Instance %s", i.Name)
	ic.enqueueInstance(i)
}

func (ic *InstanceController) updateInstance(old, cur interface{}) {
	oldI := old.(*ecsv1.Instance)
	curI := cur.(*ecsv1.Instance)
	glog.V(4).Infof("Updating Instance %s", oldI.Name)
	ic.enqueueInstance(curI)
}

func (ic *InstanceController) deleteInstance(obj interface{}) {
	i, ok := obj.(*ecsv1.Instance)
	if !ok {
		glog.Errorf("Couldn't get key for object %#v", obj)
		return
	}
	glog.V(4).Infof("Deleting Instance %s", i.Name)
	ic.enqueueInstance(i)
}

// enqueueInstance takes a Instance resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Instance.
func (ic *InstanceController) enqueueInstance(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	ic.workqueue.AddRateLimited(key)
}

// instanceCreate takes a K8s Instance resource and call OpenStack client to create an instance
func (ic *InstanceController) instanceCreate(instance *ecsv1.Instance) (string, error) {
	serverCreateOpts := servers.CreateOpts{
		Name:      instance.ObjectMeta.Name,
		ImageRef:  instance.Spec.ImageRef,
		FlavorRef: instance.Spec.FlavorRef,
		Networks: []servers.Network{
			servers.Network{UUID: instance.Spec.Network.UUID},
		},
	}

	osInstance, err := ic.osService.InstanceCreate(serverCreateOpts)
	if err != nil {
		return "", err
	}
	return osInstance.ID, nil
}

// instanceDelete takes a K8s Instance resource and call OpenStack client to force delete an instance
func (ic *InstanceController) instanceDelete(instance *ecsv1.Instance) error {
	return ic.osService.InstanceForceDelete(instance.Status.ID)
}

// instanceDelete takes a K8s Instance resource and call OpenStack client to get instance detail info
func (ic *InstanceController) instanceGet(instance *ecsv1.Instance) error {
	osInstance, err := ic.osService.InstanceGet(instance.Status.ID)
	if err != nil {
		return err
	}

	instance.Status.Status = osInstance.Status
	instance.Status.TenantID = osInstance.TenantID

	return nil
}

// processInstanceDelete takes a instance key (namespace+name) from workqueue
// and  get instance detail from local cache which contains OpenStack instance UUID
// then delete OpenStack instance and remove from local cache
func (ic *InstanceController) processInstanceDelete(key string) error {
	instanceCache, ok := ic.localCache.get(key)
	if !ok {
		return fmt.Errorf("Local cache is unconsistend with cluster cache")
	}
	err := ic.instanceDelete(instanceCache.inst)
	if err != nil {
		return err
	}
	ic.localCache.delete(key)
	return nil
}

// get a instanceCache
func (s *instanceCache) get(instanceName string) (*cachedInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instanceMap[instanceName]
	return instance, ok
}

// getOrCreate a instanceCache
func (s *instanceCache) getOrCreate(instanceName string) *cachedInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instanceMap[instanceName]
	if !ok {
		instance = &cachedInstance{}
		s.instanceMap[instanceName] = instance
	}
	return instance
}

// set a instanceCache
func (s *instanceCache) set(instanceName string, instance *cachedInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instanceMap[instanceName] = instance
}

// delete a instanceCache
func (s *instanceCache) delete(instanceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.instanceMap, instanceName)
}
