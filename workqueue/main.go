package main

import (
	"github.com/google/martian/log"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"client-go-demo/workqueue/pkg"
)

func main() {
	// 创建 config 对象
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Infof("%v", err)
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}

	// 创建 clientset
	clientset := kubernetes.NewForConfigOrDie(config)

	// 创建 Informer
	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	serviceInformer := informerFactory.Core().V1().Services()
	ingressInformer := informerFactory.Extensions().V1beta1().Ingresses()

	// 创建 Controller
	stopCh := make(chan struct{})
	c := pkg.NewController(clientset, serviceInformer, ingressInformer)

	// 启动Informer
	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)

	c.Run(stopCh)
}
