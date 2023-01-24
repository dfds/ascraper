package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/env"
	"log"
	"net/http"
	"time"
)

const DUMMY_DATA_URL = "https://petstore.swagger.io/v2/swagger.json"

func main() {
	client, err := getK8sClient()
	if err != nil {
		log.Fatal(err)
	}

	hClient := http.DefaultClient

	for {
		svcs, err := client.CoreV1().Services("selfservice").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		for _, svc := range svcs.Items {
			fmt.Println(svc.Name)

			for _, port := range svc.Spec.Ports {
				if port.Protocol == v1.ProtocolTCP {
					req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d", svc.Spec.ClusterIP, port.Port), nil)
					if err != nil {
						log.Println(err)
						continue
					}

					resp, err := hClient.Do(req)
					if err != nil {
						log.Printf("Unreachable service at %s, skipping\n", fmt.Sprintf("http://%s:%d", svc.Spec.ClusterIP, port.Port))
						continue
					}

					fmt.Println("Status code: ", resp.StatusCode)
					fmt.Println("Headers:")
					for k, v := range resp.Header {
						fmt.Printf("  %s: %v\n", k, v)
					}

				}
			}

			continue

			// Pretend this service has an openapi spec
			req, err := http.NewRequest("GET", DUMMY_DATA_URL, nil)
			if err != nil {
				log.Fatal(err)
			}

			resp, err := hClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			defer resp.Body.Close()

			rawData, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			// TODO: Discuss encoding of OpenAPI manifest
			//b64EncodedSpec := base64.StdEncoding.EncodeToString(rawData)

			// TODO: GZIP OpenAPI response
			payload := ServiceResponse{
				Name:        svc.Name,
				Namespace:   svc.Namespace,
				OpenApiSpec: string(rawData),
			}

			kMsg := Envelope[interface{}]{
				MessageId: "E3DBBBA7-E3FB-42F8-8DE2-3E3AC5E6167E",
				Type:      "placeholder",
				Data:      payload,
			}

			serialisedPayload, err := json.Marshal(kMsg)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(serialisedPayload))

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

type ServiceResponse struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	OpenApiSpec string `json:"openApiSpec"`
}

// Envelope
// Kafka message envelope/wrapper
type Envelope[D any] struct {
	MessageId string `json:"messageId"`
	Type      string `json:"type"`
	Data      D      `json:"data"`
}
