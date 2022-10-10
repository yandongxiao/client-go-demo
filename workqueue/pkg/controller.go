package pkg

import (
	"github.com/google/martian/log"
	v12 "k8s.io/client-go/informers/core/v1"
	v1beta12 "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	clientset     *kubernetes.Clientset
	serviceLister v1.ServiceLister
	ingressLister v1beta1.IngressLister
}

func (c *Controller) addService(obj interface{}) {
	log.Infof("add Service")
}

func (c *Controller) updateService(obj interface{}, obj2 interface{}) {
	log.Infof("update Service")
}

func (c *Controller) addIngress(obj interface{}) {
	log.Infof("add ingress")
}

func (c *Controller) updateIngress(obj interface{}, obj2 interface{}) {
	log.Infof("update ingress")
}

func (c *Controller) Run(stopCh chan struct{}) {
	<-stopCh
}

func NewController(clientset *kubernetes.Clientset,
	serviceInformer v12.ServiceInformer,
	ingressInformer v1beta12.IngressInformer) *Controller {

	c := &Controller{
		clientset:     clientset,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addIngress,
		UpdateFunc: c.updateIngress,
	})

	return c
}
