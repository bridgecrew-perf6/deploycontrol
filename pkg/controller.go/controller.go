package controller

import (
	"context"
	"log"
	"time"

	klientset "github.com/sahil-lakhwani/deploycontrol/pkg/client/clientset/versioned"
	customscheme "github.com/sahil-lakhwani/deploycontrol/pkg/client/clientset/versioned/scheme"
	kinf "github.com/sahil-lakhwani/deploycontrol/pkg/client/informers/externalversions/sahil.dev/v1alpha1"
	klister "github.com/sahil-lakhwani/deploycontrol/pkg/client/listers/sahil.dev/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	client kubernetes.Interface

	// clientset for custom resource kluster
	klient klientset.Interface
	// kluster has synced
	klusterSynced cache.InformerSynced
	// lister
	kLister klister.HADeploymentLister
	// queue
	wq workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewController(client kubernetes.Interface, klient klientset.Interface, informer kinf.HADeploymentInformer) *Controller {
	runtime.Must(customscheme.AddToScheme(scheme.Scheme))

	eveBroadCaster := record.NewBroadcaster()
	eveBroadCaster.StartStructuredLogging(0)
	eveBroadCaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})
	recorder := eveBroadCaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "HADeployment"})

	c := &Controller{
		client:        client,
		klient:        klient,
		klusterSynced: informer.Informer().HasSynced,
		kLister:       informer.Lister(),
		wq:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "HADeployment"),
		recorder:      recorder,
	}

	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.handleAdd,
		},
	)

	return c
}

func (c *Controller) Run(ch chan struct{}) error {
	if ok := cache.WaitForCacheSync(ch, c.klusterSynced); !ok {
		log.Println("cache was not sycned")
	}

	go wait.Until(c.worker, time.Second, ch)

	<-ch
	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	item, shutDown := c.wq.Get()
	if shutDown {
		// logs as well
		return false
	}

	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("error %s calling Namespace key func on cache for item", err.Error())
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("splitting key into namespace and name, error %s\n", err.Error())
		return false
	}

	haDeployment, err := c.kLister.HADeployments(ns).Get(name)
	if err != nil {
		log.Printf("error %s, Getting the kluster resource from lister", err.Error())
		return false
	}
	log.Printf("Spec of CR: %+v\n", haDeployment.Spec)

	labels := map[string]string{
		"app":        "nginx",
		"controller": haDeployment.Name,
	}
	d := appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      haDeployment.Name + "-deployment",
			Namespace: haDeployment.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &haDeployment.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  haDeployment.Name + "-container",
							Image: haDeployment.Spec.Image,
						},
					},
				},
			},
		},
	}

	_, err = c.client.AppsV1().Deployments(ns).Create(context.Background(), &d, metav1.CreateOptions{})
	if err != nil {
		log.Printf("error %s, creating deployment", err.Error())
		return false
	}

	c.recorder.Event(haDeployment, corev1.EventTypeNormal, "CREATE", "Deployment was created")

	return true
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd was called")
	c.wq.Add(obj)
}
