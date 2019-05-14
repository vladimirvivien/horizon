package coordinator

import (
	"fmt"
	"log"
	"time"

	"github.com/vladimirvivien/horizon/pkg/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
)

var (
	deploymentsResource = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	podsResource        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
)

type appCoordinator struct {
	k8sClient       *k8sClient
	informer        informers.GenericInformer
	informerFac     dynamicinformer.DynamicSharedInformerFactory
	coordEventFunc  api.CoordEventFunc
	podEventFunc    api.PodEventFunc
	deployEventFunc api.DeploymentEventFunc
}

func New(k8s *k8sClient) api.Coordinator {
	// return newCoord(namespace)
	return nil
}

func newCoord(k8s *k8sClient) *appCoordinator {
	factory := dynamicinformer.NewDynamicSharedInformerFactory(k8s.clientset, time.Second*3)
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
	if !syncMap[deploymentsResource] {
		return fmt.Errorf("failed to sync resource %s", deploymentsResource)
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
	ctrl := newController(c.informerFac, deploymentsResource)
	ctrl.setObjectAddedFunc(func(obj interface{}) {
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
				ReadyReplicas: getReadyReplicas(uObj),
				Source:        uObj,
			}
			c.deployEventFunc(e)
		}
	})

	ctrl.setObjectUpdatedFunc(func(old, new interface{}) {
		if c.deployEventFunc != nil {
			newOne := new.(*unstructured.Unstructured)
			newResVer, ok, err := unstructured.NestedString(newOne.Object, "metadata", "resourceversion")
			if err != nil || !ok {
				log.Println(err)
				return
			}
			oldOne := old.(*unstructured.Unstructured)
			oldResVer, ok, err := unstructured.NestedString(oldOne.Object, "metadata", "resourceversion")
			if err != nil || !ok {
				log.Println(err)
				return
			}

			// only trigger if obj different
			if newResVer != oldResVer {
				e := api.DeploymentEvent{
					Type:          api.DeploymentEventUpdate,
					Name:          newOne.GetName(),
					Namespace:     newOne.GetNamespace(),
					ReadyReplicas: getReadyReplicas(newOne),
					Source:        newOne,
				}
				c.deployEventFunc(e)
			}
		}
	})

	ctrl.setObjectDeletedFunc(func(obj interface{}) {
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
				ReadyReplicas: getReadyReplicas(uObj),
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
	ctrl := newController(c.informerFac, podsResource)
	ctrl.setObjectAddedFunc(func(obj interface{}) {
		if c.podEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}
			eType := api.PodEventNew
			phase := getPodPhase(uObj)
			if phase == "Running" {
				eType = api.PodEventRunning
			}
			e := api.PodEvent{
				Type:      eType,
				Name:      uObj.GetName(),
				Namespace: uObj.GetNamespace(),
				HostIP:    getPodHostIP(uObj),
				PodIP:     getPodIP(uObj),
			}
			c.podEventFunc(e)
		}
	})

	ctrl.setObjectUpdatedFunc(func(old, new interface{}) {
		if c.podEventFunc != nil {
			newOne := new.(*unstructured.Unstructured)
			newResVer, ok, err := unstructured.NestedString(newOne.Object, "metadata", "resourceversion")
			if err != nil || !ok {
				log.Println(err)
				return
			}
			oldOne := old.(*unstructured.Unstructured)
			oldResVer, ok, err := unstructured.NestedString(oldOne.Object, "metadata", "resourceversion")
			if err != nil || !ok {
				log.Println(err)
				return
			}

			// only trigger if obj different
			if newResVer != oldResVer {
				eType := api.PodEventUpdate
				phase := getPodPhase(newOne)
				if phase == "Running" {
					eType = api.PodEventRunning
				}
				e := api.PodEvent{
					Type:      eType,
					Name:      newOne.GetName(),
					Namespace: newOne.GetNamespace(),
					HostIP:    getPodHostIP(newOne),
					PodIP:     getPodIP(newOne),
				}
				c.podEventFunc(e)
			}
		}
	})

	ctrl.setObjectDeletedFunc(func(obj interface{}) {
		if c.podEventFunc != nil {
			uObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("unexpected type for object")
				return
			}

			e := api.PodEvent{
				Type:      api.PodEventDelete,
				Name:      uObj.GetName(),
				Namespace: uObj.GetNamespace(),
				HostIP:    getPodHostIP(uObj),
				PodIP:     getPodIP(uObj),
			}
			c.podEventFunc(e)
		}
	})
}

func getReadyReplicas(obj *unstructured.Unstructured) int64 {
	reps, ok, err := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	if !ok || err != nil {
		log.Println("failed to get deployment readyRplicas, error:", err)
	}
	return reps
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
