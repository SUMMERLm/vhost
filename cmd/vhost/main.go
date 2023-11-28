package main

import (
	"flag"
	clientset "github.com/SUMMERLm/vhost/pkg/generated/clientset/versioned"
	informers "github.com/SUMMERLm/vhost/pkg/generated/informers/externalversions"
	"github.com/SUMMERLm/vhost/pkg/signals"
	"github.com/SUMMERLm/vhost/pkg/vhostcontroller"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()
	//本地调试，上线还原
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	//cfg, err := clientcmd.BuildConfigFromFlags("", "/conf/vhost/config")
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes dynamicClient : %s", err.Error())
	}
	vhostsClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building vhost clientset: %s", err.Error())
	}

	vhostInformerFactory := informers.NewSharedInformerFactory(vhostsClient, time.Second*60)

	//controller  local and master together
	controller := pkg.NewController(kubeClient, *dynamicClient, vhostsClient,
		vhostInformerFactory.Frontends().V1().Vhosts())
	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	vhostInformerFactory.Start(stopCh)
	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
