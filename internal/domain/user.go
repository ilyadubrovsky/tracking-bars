package domain

type User struct {
	ID              int64
	BarsCredentials *BarsCredentials
	ProgressTable   *ProgressTable
}
