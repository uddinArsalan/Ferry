package db

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"errors"
	"os"
)

func NewDB() (*sql.DB, error) {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		return nil, errors.New("empty connection string")
	}
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

