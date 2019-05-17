package main

import (
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/worker"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig, ns string
	flag.StringVar(&ns, "namespace", "default", "namespace")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig file")
	flag.Parse()

	stopCh := make(chan struct{})

	var config *restclient.Config
	if len(kubeconfig) > 0 {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
		config = cfg
		log.Println("Using kubeconfig: ", kubeconfig)
	} else {
		cfg, err := restclient.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
		config = cfg
		log.Println("Using in-cluster config")
	}

	// setup the coodinator
	worker, err := worker.New("greeter-worker", ns, config)
	if err != nil {
		log.Fatalf("failed to start worker: %s", err)
	}

	worker.OnWorkerEvent(func(e api.WorkerEvent) {
		log.Println("Worker started!")
		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				io.WriteString(w, "Hello, world!\n")
			})
			log.Fatal(http.ListenAndServe(":8086", nil))
		}()
	})

	//  start coordinator
	if err := worker.Start(stopCh); err != nil {
		log.Fatal(err)
	}

	select {
	case <-stopCh:
	}
}
