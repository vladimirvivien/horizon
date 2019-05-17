package worker

import (
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/client"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	restclient "k8s.io/client-go/rest"
)

type appWorker struct {
	name            string
	k8sClient       *client.K8sClient
	informer        informers.GenericInformer
	informerFac     dynamicinformer.DynamicSharedInformerFactory
	workerEventFunc api.WorkerEventFunc
}

func New(name string, namespace string, config *restclient.Config) (api.Worker, error) {
	client, err := client.New(namespace, config)
	if err != nil {
		return nil, err
	}
	worker := newWorker(client)
	worker.name = name
	return worker, nil
}

func newWorker(k8s *client.K8sClient) *appWorker {
	factory := dynamicinformer.NewDynamicSharedInformerFactory(k8s.Interface(), time.Second*3)
	return &appWorker{k8sClient: k8s, informerFac: factory}
}

func (w *appWorker) Start(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// setup informers

	// start factory
	w.informerFac.Start(stopCh)
	//syncMap := w.informerFac.WaitForCacheSync(stopCh)

	if w.workerEventFunc != nil {
		w.workerEventFunc(api.WorkerEvent{Type: api.WorkerEventStart})
	}

	return nil
}

func (w *appWorker) OnWorkerEvent(f api.WorkerEventFunc) api.Worker {
	w.workerEventFunc = f
	return w
}
