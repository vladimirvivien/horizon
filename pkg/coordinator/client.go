package coordinator

import (
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sClient struct {
	clientset dynamic.Interface
	config    *restclient.Config
	ns        string
}

func newK8sClient(namespace string, config *restclient.Config) (*k8sClient, error) {
	cs, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = "default"
	}
	return &k8sClient{clientset: cs, config: config, ns: namespace}, nil
}

func newK8sClientFromCluster() (*k8sClient, error) {
	cfg, err := restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return newK8sClient("", cfg)
}

func newK8sClientFromConfigFile(kubeCfgPath string) (*k8sClient, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	if err != nil {
		return nil, err
	}
	return newK8sClient("", cfg)
}

func (k8s *k8sClient) get() dynamic.Interface {
	return k8s.clientset
}

func (k8s *k8sClient) namespace() string {
	return k8s.ns
}
