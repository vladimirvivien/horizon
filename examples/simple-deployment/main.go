package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/vladimirvivien/horizon/pkg/api"

	"github.com/vladimirvivien/horizon/pkg/coordinator"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var ns, image string
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	flag.StringVar(&ns, "namespace", "default", "namespace")
	flag.StringVar(&image, "image", "", "container image name")
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig file")
	flag.Parse()

	stopCh := make(chan struct{})

	log.Println("Using kubeconfig: ", kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// setup the coodinator
	coord, err := coordinator.New("coord-nginx", ns, config)
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
