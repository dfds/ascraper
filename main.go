package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/env"
	"log"
	"time"
)

func main() {
	client, err := getK8sClient()
	if err != nil {
		log.Fatal(err)
	}

	for {
		svcs, err := client.CoreV1().Services("selfservice").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		for _, svc := range svcs.Items {
			fmt.Println(svc.Name)
			fmt.Println(svc.Spec)
		}

		fmt.Println("zzzz")
		time.Sleep(time.Second * 60)
	}
}

func getK8sClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", env.GetString("KUBECONFIG", ""))
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
