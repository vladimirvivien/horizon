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

func TestCoordStart(t *testing.T) {
	tests := []struct {
		name      string
		startFunc func(api.CoordEvent)
	}{
		{
			name: "normal start",
			startFunc: func(e api.CoordEvent) {
				if e.Type != api.CoordEventStart {
					t.Error("Expecting start event")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeout := time.Duration(3 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
			coord := newCoord(client.NewFromDynamicClient("", fakeClient))

			coord.OnCoordEvent(test.startFunc)

			if err := coord.Start(ctx.Done()); err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestCoordDeploy(t *testing.T) {
	tests := []struct {
		name      string
		runParam  api.RunParam
		coordFunc func(chan struct{}, api.Coordinator, api.RunParam)
	}{
		{
			name:     "simple deployment",
			runParam: api.RunParam{Namespace: "appns", Name: "app-name", Image: "image:latest"},
			coordFunc: func(ch chan struct{}, coord api.Coordinator, param api.RunParam) {
				coord.OnDeploymentEvent(func(e api.DeploymentEvent) {
					if e.Name != param.Name {
						t.Error("unexpected value for Name", e.Name)
					}
					if e.Namespace != param.Namespace {
						t.Error("unexpected value for Name", e.Namespace)
					}
					close(ch)
				})

				if err := coord.Start(ch); err != nil {
					t.Fatal(err)
				}

				if err := coord.Run(param); err != nil {
					t.Error(err)
				}
			},
		},

		// TODO(vladimirvivien)
		// Add update and delete deployment test cases
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeout := time.Duration(3 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
			coord := newCoord(client.NewFromDynamicClient("", fakeClient))

			stopCh := make(chan struct{})
			test.coordFunc(stopCh, coord, test.runParam)
			select {
			case <-stopCh:
			case <-ctx.Done():
				t.Error("doployment test took too long")
			}
		})
	}
}

func TestCoordDeploy_WithPodEvent(t *testing.T) {
	tests := []struct {
		name      string
		runParam  api.RunParam
		coordFunc func(chan struct{}, *fake.FakeDynamicClient, api.Coordinator, api.RunParam)
	}{
		{
			name:     "simple deployment",
			runParam: api.RunParam{Namespace: "appns", Name: "app-name", Image: "image:latest"},
			coordFunc: func(ch chan struct{}, client *fake.FakeDynamicClient, coord api.Coordinator, param api.RunParam) {
				coord.OnDeploymentEvent(func(e api.DeploymentEvent) {
					// simulate pod deployment from Deployment
					pod := generateTestPod("app-name", "appns", "image:latest")
					_, err := client.Resource(podsResource).Namespace("appns").Create(pod, metav1.CreateOptions{})
					if err != nil {
						t.Error(err)
					}
				})

				coord.OnPodEvent(func(e api.PodEvent) {
					if e.Name != "app-name" {
						t.Error("unexpected pod name:", e.Name)
					}
					if e.Namespace != "appns" {
						t.Error("unexpected pod namespace:", e.Namespace)
					}
					if e.HostIP != "192.168.176.128" {
						t.Error("unexpected pod host IP value:", e.HostIP)
					}
					if e.PodIP != "172.17.0.8" {
						t.Error("unexpected pod ip value:", e.PodIP)
					}
					close(ch)
				})

				if err := coord.Start(ch); err != nil {
					t.Fatal(err)
				}

				if err := coord.Run(param); err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeout := time.Duration(3 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			fakeClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
			coord := newCoord(client.NewFromDynamicClient("", fakeClient))

			stopCh := make(chan struct{})
			test.coordFunc(stopCh, fakeClient, coord, test.runParam)
			select {
			case <-stopCh:
			case <-ctx.Done():
				t.Error("doployment test took too long")
			}
		})
	}
}

func generateTestPod(name, ns, image string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": ns,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            name,
						"image":           image,
						"imagePullPolicy": "IfNotPresent",
						"ports": []interface{}{
							map[string]interface{}{
								"containerPort": int64(8086),
							},
						},
					},
				},
			},
			"status": map[string]interface{}{
				"hostIP": "192.168.176.128",
				"phase":  "Running",
				"podIP":  "172.17.0.8",
			},
		},
	}
}
