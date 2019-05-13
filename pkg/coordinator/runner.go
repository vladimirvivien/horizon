package coordinator

import (
	"errors"

	"github.com/vladimirvivien/horizon/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (c *appCoordinator) Run(param api.RunParam) error {
	if err := assertValidRunParam(param); err != nil {
		return err
	}
	if param.Replicas == 0 {
		param.Replicas = 1
	}

	// create object
	cl := c.k8sClient.get()
	deployment := generateDeployment(param)
	_, err := cl.Resource(deploymentsResource).Namespace(param.Namespace).Create(deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func assertValidRunParam(param api.RunParam) error {
	if param.Name == "" {
		return errors.New("missing deployment name")
	}
	if param.Image == "" {
		return errors.New("missing deployment image")
	}
	return nil
}

func generateDeployment(param api.RunParam) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      param.Name,
				"namespace": param.Namespace,
			},
			"spec": map[string]interface{}{
				"replicas": param.Replicas,
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{},

					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":            param.Name,
								"image":           param.Image,
								"imagePullPolicy": "IfNotPresent",
							},
						},
					},
				},
			},
		},
	}
}
