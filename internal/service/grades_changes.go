package service

type GrandesChanges interface {
	Start() (func(), error)
	Stop() error
}
