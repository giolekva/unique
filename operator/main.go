package main

import (
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/giolekva/unique/operator/generated/clientset/versioned"
	informers "github.com/giolekva/unique/operator/generated/informers/externalversions"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

func main() {
	flag.Parse()
	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	uniqueClient := clientset.NewForConfigOrDie(cfg)
	// utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, 5*time.Second)
	uniqueInformerFactory := informers.NewSharedInformerFactory(uniqueClient, 5*time.Second)
	c := NewUniqueCountsController(
		kubeClient,
		uniqueClient,
		uniqueInformerFactory.Lekva().V1().CountUniques())
	stopCh := make(chan struct{})
	kubeInformerFactory.Start(stopCh)
	uniqueInformerFactory.Start(stopCh)
	if err := c.Run(1, stopCh); err != nil {
		panic(err)
	}
}
