package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	uniquev1 "github.com/giolekva/unique/operator/apis/unique/v1"
	clientset "github.com/giolekva/unique/operator/generated/clientset/versioned"
	informers "github.com/giolekva/unique/operator/generated/informers/externalversions/unique/v1"
	listers "github.com/giolekva/unique/operator/generated/listers/unique/v1"
)

type UniqueCountsController struct {
	kubeClient   kubernetes.Interface
	uniqueClient clientset.Interface
	uniqueLister listers.CountUniqueLister
	uniqueSynced cache.InformerSynced
	workqueue    workqueue.RateLimitingInterface
}

func NewUniqueCountsController(kubeClient kubernetes.Interface,
	uniqueClient clientset.Interface,
	uniqueInformer informers.CountUniqueInformer) *UniqueCountsController {
	c := &UniqueCountsController{
		kubeClient:   kubeClient,
		uniqueClient: uniqueClient,
		uniqueLister: uniqueInformer.Lister(),
		uniqueSynced: uniqueInformer.Informer().HasSynced,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Unique"),
	}
	uniqueInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueCountUniques,
		UpdateFunc: func(_, o interface{}) {
			c.enqueueCountUniques(o)
		},
		DeleteFunc: func(o interface{}) {
		},
	})
	return c
}

func (c *UniqueCountsController) enqueueCountUniques(o interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(o); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *UniqueCountsController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	klog.Info("Starting CountUniques controller")
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.uniqueSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}
	fmt.Println("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	fmt.Println("Started workers")
	<-stopCh
	fmt.Println("Shutting down workers")
	return nil
}

func (c *UniqueCountsController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *UniqueCountsController) processNextWorkItem() bool {
	o, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(o interface{}) error {
		defer c.workqueue.Done(o)
		if key, ok := o.(string); ok {
			if err := c.processCountUniques(key); err != nil {
				c.workqueue.AddRateLimited(key)
				return fmt.Errorf("Error syncing '%s': %s, requeuing", key, err.Error())
			}
			fmt.Printf("Successfully synced CountUniques '%s'\n", key)
		} else {
			c.workqueue.Forget(o)
			utilruntime.HandleError(fmt.Errorf("expected reference in workqueue but got %#v", o))
			return nil
		}
		c.workqueue.Forget(o)
		return nil
	}(o)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *UniqueCountsController) processCountUniques(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	cu, err := c.uniqueLister.CountUniques(namespace).Get(name)
	if err != nil {
		panic(err)
	}
	if cu.Status.State == uniquev1.CountUniqueStateRunning {
		fmt.Printf("%s CountUnique is already in Running state\n", cu.Name)
		return nil
	}
	c.updateCountUniqueStatus(cu, uniquev1.CountUniqueStateRunning, "olalaaaa")
	return nil
}

func (c *UniqueCountsController) updateCountUniqueStatus(cu *uniquev1.CountUnique, state uniquev1.CountUniqueState, msg string) error {
	cp := cu.DeepCopy()
	cp.Status.State = state
	cp.Status.Message = msg
	_, err := c.uniqueClient.LekvaV1().CountUniques(cp.Namespace).UpdateStatus(context.TODO(), cp, metav1.UpdateOptions{})
	return err
}
