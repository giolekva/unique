package main

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

var ctrlParallelism int32 = 1

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
	_, err = c.kubeClient.BatchV1().Jobs(cu.Namespace).Create(context.TODO(), c.generateControllerConfig(cu), metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = c.kubeClient.CoreV1().Services(cu.Namespace).Create(context.TODO(), c.generateServiceConfig(cu), metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = c.kubeClient.BatchV1().Jobs(cu.Namespace).Create(context.TODO(), c.generateWorkersConfig(cu), metav1.CreateOptions{})
	if err != nil {
		return err
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

func (c *UniqueCountsController) generateControllerConfig(cu *uniquev1.CountUnique) *batchv1.Job {
	ctrlName := fmt.Sprintf("%s-controller", cu.Name)
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrlName,
			Namespace: cu.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &ctrlParallelism,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": ctrlName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "controller",
							Image:           "giolekva/unique:v0.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									Name:          "controller",
									ContainerPort: 4321,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Command: []string{
								"server_controller",
								fmt.Sprintf("--num-bits=%d", cu.Spec.NumBits),
								"--port=4321",
								fmt.Sprintf("--start-from=%s", cu.Spec.StartFrom),
								fmt.Sprintf("--num-documents=%d", cu.Spec.NumDocuments),
							},
						},
					},
				},
			},
		},
	}
}

func (c *UniqueCountsController) generateWorkersConfig(cu *uniquev1.CountUnique) *batchv1.Job {
	ctrlName := fmt.Sprintf("%s-controller", cu.Name)
	workersName := fmt.Sprintf("%s-workers", cu.Name)
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workersName,
			Namespace: cu.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &cu.Spec.NumWorkers,
			Completions: &cu.Spec.NumWorkers,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": workersName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "worker",
							Image:           "giolekva/unique:v0.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								corev1.EnvVar{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									Name:          "worker",
									ContainerPort: 1234,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Command: []string{
								"server_worker",
								fmt.Sprintf("--controller-address=%s.%s.svc.cluster.local:8080", ctrlName, cu.Namespace),
								fmt.Sprintf("--num-workers=%d", cu.Spec.WorkerParallelism),
								fmt.Sprintf("--num-bits=%d", cu.Spec.NumBits),
								"--port=1234",
								"--name=$(POD_NAME)",
								"--address=$(POD_IP)",
							},
						},
					},
				},
			},
		},
	}
}

func (c *UniqueCountsController) generateServiceConfig(cu *uniquev1.CountUnique) *corev1.Service {
	ctrlName := fmt.Sprintf("%s-controller", cu.Name)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrlName,
			Namespace: cu.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": ctrlName,
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:       "controller",
					Port:       8080,
					TargetPort: intstr.FromString("controller"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
