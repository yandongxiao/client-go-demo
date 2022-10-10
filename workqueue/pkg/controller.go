package pkg

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v1core "k8s.io/api/core/v1"
	v1networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v1coreinformer "k8s.io/client-go/informers/core/v1"
	v1networkinginformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	v1corelister "k8s.io/client-go/listers/core/v1"
	v1networkinglister "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	clientset     *kubernetes.Clientset
	serviceLister v1corelister.ServiceLister
	ingressLister v1networkinglister.IngressLister
	queue         workqueue.RateLimitingInterface
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *Controller) updateService(oldObj interface{}, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *Controller) deleteIngress(obj interface{}) {
	ingress, ok := obj.(*v1networking.Ingress)
	if !ok {
		runtime.HandleError(fmt.Errorf("this is not ingress: %v", obj))
		return
	}

	// 获取OwnerReference, 注意对ownerReference的判断
	ownerReference := metav1.GetControllerOf(ingress)
	if ownerReference == nil || ownerReference.Kind != "Service" {
		return
	}

	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *Controller) Run(stopCh chan struct{}) {
	for i := 0; i < 5; i++ {
		// wait.Until 处理了信号 stopCh
		// 如果 c.worker 异常退出后， wait.Until 能保证再次启动一个新的 worker
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

func (c *Controller) worker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		// HandleError 底层到底都做了什么？是否需要 return 语句？
		runtime.HandleError(fmt.Errorf("workqueue is shutdown"))
		return false
	}
	defer c.queue.Done(item)

	if err := c.reconcileService(item); err != nil {
		runtime.HandleError(err)
		if c.queue.NumRequeues(item) > 10 {
			c.queue.Forget(item)
		} else {
			c.queue.AddRateLimited(item)
		}
	}
	return true
}

func (c *Controller) reconcileService(item interface{}) error {
	ctx := context.Background()

	key := item.(string)
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// 获取 service 对象
	service, err := c.serviceLister.Services(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// 删除对应的 ingress 对象
			if err := c.clientset.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{});
				err != nil && !errors.IsNotFound(err) {
				return err
			}
			return nil
		}
		return err
	}

	// 检查 service 对象是否存在 Annotation
	if _, ok := service.Annotations["ingress/http"]; !ok {
		return nil
	}

	expectIngress := c.constructIngress(service)

	// 检查同名的 Ingress 对象是否存在？
	ingress, err := c.ingressLister.Ingresses(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err := c.clientset.NetworkingV1().Ingresses(ns).Create(ctx, expectIngress, metav1.CreateOptions{});
				err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// 如何比较实际状态和期望状态？尤其是实际状态包括了Status部分，Meta部分也有额外的内容。
	if reflect.DeepEqual(expectIngress.Spec, ingress.Spec) {
		return nil
	}

	if c.clientset.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	_, err = c.clientset.NetworkingV1().Ingresses(ns).Create(ctx, expectIngress, metav1.CreateOptions{})
	return err
}

func NewController(clientset *kubernetes.Clientset,
	serviceInformer v1coreinformer.ServiceInformer,
	ingressInformer v1networkinginformer.IngressInformer) *Controller {

	c := &Controller{
		clientset:     clientset,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "service"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}

func (c *Controller) constructIngress(service *v1core.Service) *v1networking.Ingress {
	ingress := v1networking.Ingress{}

	ingress.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(service, v1core.SchemeGroupVersion.WithKind("Service")),
	}

	ingress.Name = service.Name
	ingress.Namespace = service.Namespace
	pathType := v1networking.PathTypePrefix
	icn := "nginx"
	ingress.Spec = v1networking.IngressSpec{
		IngressClassName: &icn,
		Rules: []v1networking.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: v1networking.IngressRuleValue{
					HTTP: &v1networking.HTTPIngressRuleValue{
						Paths: []v1networking.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: v1networking.IngressBackend{
									Service: &v1networking.IngressServiceBackend{
										Name: service.Name,
										Port: v1networking.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &ingress
}
