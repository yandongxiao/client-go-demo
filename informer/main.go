package main

import (
	"fmt"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	clientset := kubernetes.NewForConfigOrDie(config)

	// 选择使用 WithOptions 的 New 方法是因为：我们希望指定Namespace。
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace("kube-system"))
	// 本质是调用 factory 的 InformerFor 方法，创建一个 Informer。
	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("Add Event")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("Update Event")
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("Delete Event")
		},
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)

	factory.WaitForCacheSync(stopCh)

	select {}
}
