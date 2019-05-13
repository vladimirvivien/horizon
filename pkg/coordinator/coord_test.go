package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
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

			client := fake.NewSimpleDynamicClient(runtime.NewScheme())
			coord := newCoord(&k8sClient{clientset: client})

			coord.OnCoordEvent(test.startFunc)

			if err := coord.Start(ctx.Done()); err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestCoordDeploy(t *testing.T) {
	tests := []struct {
		name       string
		runParam   api.RunParam
		eventFunc  func(api.DeploymentEvent)
		runnerFunc func(api.Coordinator, api.RunParam)
	}{
		{
			name:     "simple deployment",
			runParam: api.RunParam{Namespace: "appns", Name: "app-name", Image: "test.app.image:latest"},
			eventFunc: func(e api.DeploymentEvent) {
				t.Log("((( deployment event rcvd )))")
				// switch e.Type {
				// case api.DeploymentEventNew:
				// 	t.Log("((( new deployment )))")
				// default:
				// 	t.Error("wrong deployment event type", e.Type)
				// }
			},
			runnerFunc: func(coord api.Coordinator, param api.RunParam) {
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

			client := fake.NewSimpleDynamicClient(runtime.NewScheme())
			coord := newCoord(&k8sClient{clientset: client})

			coord.OnDeploymentEvent(test.eventFunc)

			go func() {
				if err := coord.Start(ctx.Done()); err != nil {
					t.Fatal(err)
				}
			}()

			if test.runnerFunc != nil {
				test.runnerFunc(coord, test.runParam)
			}
		})
	}
}
