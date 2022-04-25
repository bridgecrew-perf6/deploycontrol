package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	klient "github.com/sahil-lakhwani/deploycontrol/pkg/client/clientset/versioned"
	kInfFac "github.com/sahil-lakhwani/deploycontrol/pkg/client/informers/externalversions"
	"github.com/sahil-lakhwani/deploycontrol/pkg/controller.go"
	// "github.com/sahil-lakhwani/deploycontrol/pkg/controller"
)

func main() {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Building config from flags failed, %s, trying to build inclusterconfig", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("error %s building inclusterconfig", err.Error())
		}
	}

	klientset, err := klient.NewForConfig(config)
	if err != nil {
		log.Printf("getting klient set %s\n", err.Error())
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("getting std client %s\n", err.Error())
	}

	infoFactory := kInfFac.NewSharedInformerFactory(klientset, 20*time.Minute)
	ch := make(chan struct{})
	c := controller.NewController(client, klientset, infoFactory.Sahil().V1alpha1().HADeployments())

	infoFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("error running controller %s\n", err.Error())
	}
}
