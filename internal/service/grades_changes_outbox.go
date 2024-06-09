package service

type GrandesChangesOutbox interface {
	Start()
	Stop() error
}
