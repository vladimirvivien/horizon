package api

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RunParam struct {
	Namespace       string
	Name            string
	Image           string
	ImagePullPolicy string
	Port            int64
	Envs            []string
	Labels          string
	Replicas        int64
}

type EventFunc func()

type CoordEventType int

const (
	CoordEventUnknown CoordEventType = iota
	CoordEventStart
	CoordEventStop
)

type CoordEvent struct {
	Type CoordEventType
}

type CoordEventFunc func(CoordEvent)

type PodEvent struct{}

type DeploymentEventType int

const (
	DeploymentEventUnknown DeploymentEventType = iota
	DeploymentEventNew
	DeploymentEventUpdate
	DeploymentEventDelete
)

type DeploymentEvent struct {
	Type      DeploymentEventType
	Name      string
	Namespace string
	Port      int64
	Status    string
	Source    *unstructured.Unstructured
}

type PodEventFunc func(PodEvent)
type DeploymentEventFunc func(DeploymentEvent)

type Coordinator interface {
	Start(<-chan struct{}) error
	Run(RunParam) error
	OnCoordEvent(CoordEventFunc) Coordinator
	OnPodEvent(PodEventFunc) Coordinator
	OnDeploymentEvent(DeploymentEventFunc) Coordinator
}
