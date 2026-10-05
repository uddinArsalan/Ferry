package repository

import (
	"context"
	"database/sql"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) CreateGroup(ctx context.Context, name string) (int64, error) {
	query := `INSERT INTO groups (name) VALUES ($1) RETURNING id`
	var groupID int64
	row := r.db.QueryRow(query, name)
	if err := row.Scan(&groupID); err != nil {
		return -1, err
	}
	return groupID, nil
}
