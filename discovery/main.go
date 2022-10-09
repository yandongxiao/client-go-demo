package main

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config)

	fmt.Println(discoveryClient.ServerVersion())
	fmt.Println(discoveryClient.ServerPreferredResources())
}
