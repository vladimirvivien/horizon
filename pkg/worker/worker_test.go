package worker

import (
	"context"
	"testing"
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestWorkerStart(t *testing.T) {
	tests := []struct {
		name      string
		startFunc func(api.WorkerEvent)
	}{
		{
			name: "normal start",
			startFunc: func(e api.WorkerEvent) {
				if e.Type != api.WorkerEventStart {
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
			worker := newWorker(client.NewFromDynamicClient("", fakeClient))

			stopCh := make(chan struct{})

			worker.OnWorkerEvent(test.startFunc)
			if err := worker.Start(stopCh); err != nil {
				t.Fatal(err)
			}
			select {
			case <-stopCh:
			case <-ctx.Done():
				t.Error("doployment test took too long")
			}
		})
	}
}
