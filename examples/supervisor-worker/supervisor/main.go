package main

import (
	"flag"
	"log"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/coordinator"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig, ns, image string
	flag.StringVar(&ns, "namespace", "default", "namespace")
	flag.StringVar(&image, "worker-image", image, "container image for worker process")
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
	coord, err := coordinator.New("coordinator", ns, config)
	if err != nil {
		log.Fatalf("failed to start coordinator: %s", err)
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

	//  start coordinator
	if err := coord.Start(stopCh); err != nil {
		log.Fatal(err)
	}

	// apply an operation
	if err := coord.Run(api.RunParam{Name: image, Namespace: ns, Image: image}); err != nil {
		log.Fatal(err)
	}

	select {
	case <-stopCh:
	}
}
