package api

type AppConfg string

type Coordinator interface {
	Start(name string) error
	Stop() error
	Run(AppConfg) error
}
