package client

import (
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
)

type K8sClient struct {
	clientset dynamic.Interface
	ns        string
}

func New(namespace string, config *restclient.Config) (*K8sClient, error) {
	cs, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = "default"
	}
	return NewFromDynamicClient(namespace, cs), nil
}

func NewFromDynamicClient(namespace string, client dynamic.Interface) *K8sClient {
	if namespace == "" {
		namespace = "default"
	}
	return &K8sClient{clientset: client, ns: namespace}
}

func (k8s *K8sClient) Interface() dynamic.Interface {
	return k8s.clientset
}

func (k8s *K8sClient) Namespace() string {
	return k8s.ns
}
