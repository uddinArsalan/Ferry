package repository

import (
	"context"
	"database/sql"

	"github.com/uddinArsalan/ferry-registry/internals/domain"
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

func (r *GroupRepository) GetGroupsForUser(ctx context.Context, userID int64) ([]*domain.Group, error) {
	query := "SELECT id, name, user_id, created_at FROM groups WHERE user_id = $1"
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*domain.Group
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var group domain.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	return groups, nil
}

func (r *GroupRepository) GetGroupByID(ctx context.Context, groupID int64) (*domain.Group, error) {
	query := "SELECT id, name, user_id, created_at FROM groups WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, groupID)
	var group domain.Group
	if err := row.Scan(&group.ID, &group.Name, &group.UserID, &group.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No group found with the given ID
		}
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) GetGroupsForPeer(ctx context.Context, peerID int64) ([]*domain.Group, error) {
	query := `SELECT id,name,user_id,created_at 
				FROM groups AS g
				 JOIN group_peer_membership AS m
				   ON g.id = m.group_id
				     WHERE m.peer_id = $1`
	rows, err := r.db.QueryContext(ctx, query, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*domain.Group
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var group domain.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	return groups, nil
}
