package progress_tables

import "github.com/ilyadubrovsky/tracking-bars/internal/database"

type repo struct {
	db database.PG
}

func NewRepository(db database.PG) *repo {
	return &repo{
		db: db,
	}
}
