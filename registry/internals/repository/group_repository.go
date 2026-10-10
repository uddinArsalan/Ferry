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

func (r *GroupRepository) CreateGroup(ctx context.Context, grpId, name string) (int64, error) {
	query := `INSERT INTO groups (name,group_id) VALUES ($1,$2) RETURNING id`
	// this represent db table pk
	var groupID int64
	row := r.db.QueryRowContext(ctx, query, name, grpId)
	if err := row.Scan(&groupID); err != nil {
		return -1, err
	}
	return groupID, nil
}

func (r *GroupRepository) GetGroupsForUser(ctx context.Context, userID int64) ([]*domain.Group, error) {
	query := "SELECT id, name, user_id,group_id, created_at FROM groups WHERE user_id = $1"
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
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	return groups, nil
}

func (r *GroupRepository) GetGroupByGrpID(ctx context.Context, groupID string) (*domain.Group, error) {
	query := "SELECT id, name, user_id,group_id, created_at FROM groups WHERE group_id = $1"
	row := r.db.QueryRowContext(ctx, query, groupID)
	var group domain.Group
	if err := row.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No group found with the given ID
		}
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) GetGroupsForPeer(ctx context.Context, localPeerID string) ([]*domain.Group, error) {
	// peerID - string is local id used
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var peerID int64
	query1 := `SELECT id FROM peers WHERE peer_id = $1`
	row := tx.QueryRowContext(ctx, query1, localPeerID)
	if err = row.Scan(&peerID); err != nil {
		return nil, err
	}
	query := `SELECT id,name,user_id,group_id,created_at 
				FROM groups AS g
				 JOIN group_peer_membership AS m
				   ON g.id = m.group_id
				     WHERE m.peer_id = $1`
	rows, err := tx.QueryContext(ctx, query, peerID)
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
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return groups, nil
}
