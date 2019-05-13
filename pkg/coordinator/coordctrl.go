package coordinator

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type objectAddedEventFunc func(obj interface{})
type objectUpdatedEventFunc func(old, new interface{})
type objectDeletedEventFunc func(obj interface{})

type controller struct {
	factory      dynamicinformer.DynamicSharedInformerFactory
	informer     informers.GenericInformer
	resource     schema.GroupVersionResource
	handlerFuncs *cache.ResourceEventHandlerFuncs
}

func newController(informerFactory dynamicinformer.DynamicSharedInformerFactory, res schema.GroupVersionResource) *controller {
	informer := informerFactory.ForResource(res)
	handlers := &cache.ResourceEventHandlerFuncs{}
	informer.Informer().AddEventHandler(handlers)
	return &controller{factory: informerFactory, resource: res, informer: informer, handlerFuncs: handlers}
}

func (c *controller) setObjectAddedFunc(fn objectAddedEventFunc) *controller {
	c.handlerFuncs.AddFunc = fn
	return c
}

func (c *controller) setObjectUpdatedFunc(fn objectUpdatedEventFunc) *controller {
	c.handlerFuncs.UpdateFunc = fn
	return c
}

func (c *controller) setObjectDeletedFunc(fn objectDeletedEventFunc) *controller {
	c.handlerFuncs.DeleteFunc = fn
	return c
}
