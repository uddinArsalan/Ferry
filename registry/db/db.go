package db

import (
	"database/sql"
	"errors"
	"os"
)

func NewDB() (*sql.DB, error) {
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		return nil, errors.New("empty connection string")
	}
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

