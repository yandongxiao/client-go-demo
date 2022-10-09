package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}

	var pod v1.Pod
	if err := restClient.Get().Namespace("kube-system").Resource("pods").Name("traefik-7cd4fcff68-qzgxk").Do(context.Background()).Into(&pod);
		err != nil {
		panic(err)
	}

	fmt.Println(pod.UID)
}
