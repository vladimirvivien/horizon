package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestRunner(t *testing.T) {
	tests := []struct {
		name  string
		param api.RunParam
	}{
		{
			name:  "run with default replica",
			param: api.RunParam{Namespace: "appns", Name: "app-name", Image: "test.app.image:latest"},
		},
		{
			name:  "run with replica specified",
			param: api.RunParam{Namespace: "appns", Name: "app-name", Image: "test.app.image:latest", Replicas: 3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeout := time.Duration(3 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme())

			coord := newCoord(client.NewFromDynamicClient("", fakeClient))
			if err := coord.Start(ctx.Done()); err != nil {
				t.Fatal(err)
			}

			if err := coord.Run(test.param); err != nil {
				t.Fatal(err)
			}

			// validate creation
			savedObj, err := fakeClient.Resource(deploymentsResource).Namespace(test.param.Namespace).Get(test.param.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if savedObj.GetName() != test.param.Name {
				t.Error("unexpected deployment name:", savedObj.GetName())
			}

			if savedObj.GetNamespace() != test.param.Namespace {
				t.Error("unexpected deployment namespace:", savedObj.GetNamespace())
			}
			replicas, ok, err := unstructured.NestedInt64(savedObj.Object, "spec", "replicas")
			if err != nil || !ok {
				t.Errorf("failed to get replica count: %s", err)
			}
			if (test.param.Replicas > 0 && replicas != test.param.Replicas) && (test.param.Replicas == 0 && replicas != 1) {
				t.Error("unexpected replica count: ", replicas)
			}
		})
	}
}
