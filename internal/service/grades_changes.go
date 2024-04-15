package service

type GrandesChanges interface {
	Start()
	Stop() error
}
