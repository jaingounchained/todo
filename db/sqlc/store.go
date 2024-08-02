package db

import (
	"database/sql"
)

// Store provides all functions to execute db queries and transaction
type Store interface {
	Querier
}

// SQLStore provides all functions to execute SQL queries and transaction
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}
