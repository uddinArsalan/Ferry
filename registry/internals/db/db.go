package db

import (
	"database/sql"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
