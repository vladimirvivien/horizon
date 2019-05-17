package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/coordinator"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig, ns, image string
	flag.StringVar(&ns, "namespace", "default", "namespace")
	flag.StringVar(&image, "worker-image", "worker:latest", "container image for worker process")
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
	coord, err := coordinator.New("greeter-supervisor", ns, config)
	if err != nil {
		log.Fatalf("failed to start greeter-supervisor: %s", err)
	}

	// describe callback
	coord.OnDeploymentEvent(func(e api.DeploymentEvent) {
		switch e.Type {
		case api.DeploymentEventNew:
			log.Printf("Deployment \"%s\" received\n", e.Name)
		case api.DeploymentEventUpdate:
			if e.Ready {
				log.Printf("Deployment \"%s\" ready!\n", e.Name)
			}
		}
	})

	coord.OnPodEvent(func(e api.PodEvent) {
		log.Println("Rcvd pod event")
		if e.Running {
			addr := e.PodIP
			res, err := http.Get(fmt.Sprintf("http://%s:%d/", addr, e.Port))
			if err != nil {
				log.Println("unable to connect to worker process:", err)
				return
			}
			msg, err := ioutil.ReadAll(res.Body)
			defer res.Body.Close()
			if err != nil {
				log.Println("failed to read message from worker:", err)
				return
			}
			log.Println(msg)
		}
	})

	//  start coordinator
	if err := coord.Start(stopCh); err != nil {
		log.Fatal(err)
	}

	// apply an operation
	if err := coord.Run(api.RunParam{
		Replicas:        1,
		Name:            "worker",
		Namespace:       ns,
		Image:           image,
		Port:            8086,
		ImagePullPolicy: "Never",
	}); err != nil {
		log.Fatal(err)
	}

	select {
	case <-stopCh:
	}
}
