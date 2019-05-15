package controller

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type ObjectAddedEventFunc func(obj interface{})
type ObjectUpdatedEventFunc func(old, new interface{})
type ObjectDeletedEventFunc func(obj interface{})

type Controller struct {
	factory      dynamicinformer.DynamicSharedInformerFactory
	informer     informers.GenericInformer
	resource     schema.GroupVersionResource
	handlerFuncs *cache.ResourceEventHandlerFuncs
}

func New(informerFactory dynamicinformer.DynamicSharedInformerFactory, res schema.GroupVersionResource) *Controller {
	informer := informerFactory.ForResource(res)
	handlers := &cache.ResourceEventHandlerFuncs{}
	informer.Informer().AddEventHandler(handlers)
	return &Controller{factory: informerFactory, resource: res, informer: informer, handlerFuncs: handlers}
}

func (c *Controller) SetObjectAddedFunc(fn ObjectAddedEventFunc) *Controller {
	c.handlerFuncs.AddFunc = fn
	return c
}

func (c *Controller) SetObjectUpdatedFunc(fn ObjectUpdatedEventFunc) *Controller {
	c.handlerFuncs.UpdateFunc = fn
	return c
}

func (c *Controller) SetObjectDeletedFunc(fn ObjectDeletedEventFunc) *Controller {
	c.handlerFuncs.DeleteFunc = fn
	return c
}
