package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go.dfds.cloud/scraper/internal/config"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/env"
)

var (
	SucceededScanReq = prometheus.NewGauge(prometheus.GaugeOpts{
		Help: "Successful request to a service port",
		Name: "ascraper_secceeded_scan_req",
	})

	FailedScanReq = prometheus.NewGauge(prometheus.GaugeOpts{
		Help: "Failed request to a service port",
		Name: "ascraper_failed_scan_req",
	})
)

func init() {
	prometheus.MustRegister(SucceededScanReq)
	prometheus.MustRegister(FailedScanReq)
}

const DUMMY_DATA_URL = "https://petstore.swagger.io/v2/swagger.json"

func NewDialer(authConfig AuthConfig) *kafka.Dialer {
	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// Configure SASL
	saslMechanism := plain.Mechanism{
		Username: authConfig.Username,
		Password: authConfig.Password,
	}

	// Configure connection dialer
	return &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		TLS:           tlsConfig,
		SASLMechanism: saslMechanism,
	}
}

// ConsumerConfig allows one to configure a Kafka consumer using
// environment variables.
type ConsumerConfig struct {
	GroupID string `envconfig:"group_id",required:"true"`
	Topic   string `required:"true"`
}

// ProducerConfig allows one to configure a Kafka producer using
// environment variables.
type ProducerConfig struct {
	Topic string `required:"true"`
}

// AuthConfig allows one to configure auth with a plain SASL
// authnetication mechanism to the Kafka brokers.
type AuthConfig struct {
	Brokers  []string `required:"true"`
	Username string   `required:"true"`
	Password string   `required:"true"`
}

func NewProducer(config ProducerConfig, authConfig AuthConfig, dialer *kafka.Dialer) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  authConfig.Brokers,
		Topic:    config.Topic,
		Balancer: &kafka.Hash{},
		Async:    false,
		Dialer:   dialer,
		// Not utilizing the internal retry logic of this client, since we want to keep trying
		// indefinitely on these type of errors.
		MaxAttempts:  1,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
}

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":8081", nil)
	}()
	// Initiate producers
	pconf := ProducerConfig{Topic: conf.Kafka.Topic}
	authconf := AuthConfig{Brokers: []string{conf.Kafka.Broker}, Username: conf.Kafka.Username, Password: conf.Kafka.Password}
	dialer := NewDialer(authconf)
	producer := NewProducer(pconf, authconf, dialer)
	ctx := context.Background()

	defer producer.Close()

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
					req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/swagger/v1/swagger.json", svc.Spec.ClusterIP, port.Port), nil)
					if err != nil {
						log.Println(err)
						continue
					}

					resp, err := hClient.Do(req)
					if err != nil {
						FailedScanReq.Inc()
						log.Printf("Unreachable service at %s, skipping\n", fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port.Port))
						continue
					}

					if resp.StatusCode == 200 {
						rawData, err := io.ReadAll(resp.Body)
						if err != nil {
							log.Fatal(err)
						}
						SucceededScanReq.Inc()
						defer resp.Body.Close()

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

						testMsgKey := "weeeeee" //fine for now
						err = producer.WriteMessages(ctx, kafka.Message{Key: []byte(testMsgKey), Value: serialisedPayload})
						if err != nil {
							fmt.Print(err)
							os.Exit(1)
						}
					}

				}
			}

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
