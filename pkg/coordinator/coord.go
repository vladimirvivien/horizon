package coordinator

import (
	"fmt"
	"log"
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
	"github.com/vladimirvivien/horizon/pkg/client"
	"github.com/vladimirvivien/horizon/pkg/controller"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	restclient "k8s.io/client-go/rest"
)

type appCoordinator struct {
	name            string
	k8sClient       *client.K8sClient
	informer        informers.GenericInformer
	informerFac     dynamicinformer.DynamicSharedInformerFactory
	coordEventFunc  api.CoordEventFunc
	podEventFunc    api.PodEventFunc
	deployEventFunc api.DeploymentEventFunc
}

func New(name string, namespace string, config *restclient.Config) (api.Coordinator, error) {
	client, err := client.New(namespace, config)
	if err != nil {
		return nil, err
	}
	coord := newCoord(client)
	coord.name = name
	return coord, nil
}

func newCoord(k8s *client.K8sClient) *appCoordinator {
	factory := dynamicinformer.NewDynamicSharedInformerFactory(k8s.Interface(), time.Second*3)
	// 1. setup informer/watcher for cluster wide resources (node, etc)
	// 2. Register callbacks for cluster events
	return &appCoordinator{k8sClient: k8s, informerFac: factory}
}

func (c *appCoordinator) Start(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// setup informers
	c.setupDeploymentInformer()
	c.setupPodInformer()

	// start factory
	c.informerFac.Start(stopCh)

	syncMap := c.informerFac.WaitForCacheSync(stopCh)

	// validate all resources are sync'd
	if !syncMap[api.DeploymentsResource] {
		return fmt.Errorf("failed to sync resource %s", api.DeploymentsResource)
	}

	if !syncMap[api.PodsResource] {
		return fmt.Errorf("failed to sync resource %s", api.PodsResource)
	}

	if c.coordEventFunc != nil {
		c.coordEventFunc(api.CoordEvent{Type: api.CoordEventStart})
	}

	return nil
}

func (c *appCoordinator) OnCoordEvent(e api.CoordEventFunc) api.Coordinator {
	c.coordEventFunc = e
	return c
}

func (c *appCoordinator) OnDeploymentEvent(e api.DeploymentEventFunc) api.Coordinator {
	c.deployEventFunc = e
	return c
}

func (c *appCoordinator) setupDeploymentInformer() {
	ctrl := controller.New(c.informerFac, api.DeploymentsResource)
	ctrl.SetObjectAddedFunc(func(obj interface{}) {
		if c.deployEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}
			e := api.DeploymentEvent{
				Type:          api.DeploymentEventNew,
				Name:          uObj.GetName(),
				Namespace:     uObj.GetNamespace(),
				ReadyReplicas: getDeploymentReplicasField(uObj, "readyReplicas"),
				Ready:         isDeploymentReady(uObj),
				Source:        uObj,
			}
			c.deployEventFunc(e)
		}
	})

	ctrl.SetObjectUpdatedFunc(func(old, new interface{}) {
		if c.deployEventFunc != nil {
			newOne := new.(*unstructured.Unstructured)
			newResVer, ok, err := unstructured.NestedString(newOne.Object, "metadata", "resourceVersion")
			if err != nil || !ok {
				log.Println(err)
				return
			}
			oldOne := old.(*unstructured.Unstructured)
			oldResVer, ok, err := unstructured.NestedString(oldOne.Object, "metadata", "resourceVersion")
			if err != nil || !ok {
				log.Println(err)
				return
			}

			if newResVer != oldResVer {
				e := api.DeploymentEvent{
					Type:          api.DeploymentEventUpdate,
					Name:          newOne.GetName(),
					Namespace:     newOne.GetNamespace(),
					ReadyReplicas: getDeploymentReplicasField(newOne, "readyReplicas"),
					Ready:         isDeploymentReady(newOne),
					Source:        newOne,
				}
				c.deployEventFunc(e)
			}
		}
	})

	ctrl.SetObjectDeletedFunc(func(obj interface{}) {
		if c.deployEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}

			e := api.DeploymentEvent{
				Type:          api.DeploymentEventDelete,
				Name:          uObj.GetName(),
				Namespace:     uObj.GetNamespace(),
				ReadyReplicas: getDeploymentReplicasField(uObj, "readyReplicas"),
				Ready:         isDeploymentReady(uObj),
				Source:        uObj,
			}
			c.deployEventFunc(e)
		}
	})
}

func (c *appCoordinator) OnPodEvent(e api.PodEventFunc) api.Coordinator {
	c.podEventFunc = e
	return c
}

func (c *appCoordinator) setupPodInformer() {
	ctrl := controller.New(c.informerFac, api.PodsResource)
	ctrl.SetObjectAddedFunc(func(obj interface{}) {
		if c.podEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}
			e := api.PodEvent{
				Type:      api.PodEventNew,
				Name:      uObj.GetName(),
				Namespace: uObj.GetNamespace(),
			}
			c.podEventFunc(e)
		}
	})

	ctrl.SetObjectUpdatedFunc(func(old, new interface{}) {
		if c.podEventFunc != nil {
			newOne := new.(*unstructured.Unstructured)
			newResVer, ok, err := unstructured.NestedString(newOne.Object, "metadata", "resourceVersion")
			if err != nil || !ok {
				log.Println(err)
				return
			}
			oldOne := old.(*unstructured.Unstructured)
			oldResVer, ok, err := unstructured.NestedString(oldOne.Object, "metadata", "resourceVersion")
			if err != nil || !ok {
				log.Println(err)
				return
			}

			// only trigger if obj different
			if newResVer != oldResVer {
				phase := getPodPhase(newOne)
				running := (phase == "Running")
				e := api.PodEvent{
					Type:      api.PodEventUpdate,
					Name:      newOne.GetName(),
					Namespace: newOne.GetNamespace(),
				}
				if running {
					log.Printf("Pod %s is running\n", e.Name)
					e.HostIP = getPodHostIP(newOne)
					e.PodIP = getPodIP(newOne)
					e.Running = running
				}
				c.podEventFunc(e)
			}
		}
	})

	ctrl.SetObjectDeletedFunc(func(obj interface{}) {
		if c.podEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}
			phase := getPodPhase(uObj)
			e := api.PodEvent{
				Type:      api.PodEventDelete,
				Name:      uObj.GetName(),
				Namespace: uObj.GetNamespace(),
				HostIP:    getPodHostIP(uObj),
				PodIP:     getPodIP(uObj),
				Running:   (phase == "Running"),
			}
			c.podEventFunc(e)
		}
	})
}

func getDeploymentReplicasField(obj *unstructured.Unstructured, field string) int64 {
	reps, ok, err := unstructured.NestedInt64(obj.Object, "status", field)
	if !ok || err != nil {
		return 0
	}
	return reps
}

func isDeploymentReady(obj *unstructured.Unstructured) bool {
	requestedReplicas := getDeploymentReplicasField(obj, "replicas")
	readyReplicas := getDeploymentReplicasField(obj, "readyReplicas")
	return readyReplicas == requestedReplicas
}

func getPodPhase(obj *unstructured.Unstructured) string {
	phase, ok, err := unstructured.NestedString(obj.Object, "status", "phase")
	if !ok || err != nil {
		log.Println("failed to get phase from pod status, error:", err)
		phase = "unknown"
	}
	return phase
}

func getPodHostIP(obj *unstructured.Unstructured) string {
	ip, ok, err := unstructured.NestedString(obj.Object, "status", "hostIP")
	if !ok || err != nil {
		log.Println("failed to get hostIP from pod status, error:", err)
		ip = "unknown"
	}
	return ip
}

func getPodIP(obj *unstructured.Unstructured) string {
	ip, ok, err := unstructured.NestedString(obj.Object, "status", "podIP")
	if !ok || err != nil {
		log.Println("failed to get podIP from pod status, error:", err)
		ip = "unknown"
	}
	return ip
}
