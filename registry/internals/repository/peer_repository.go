package repository

import (
	"context"
	"database/sql"
)

type PeerRepository struct {
	db *sql.DB
}

func NewPeerRepository(db *sql.DB) *PeerRepository {
	return &PeerRepository{
		db: db,
	}
}

func (r *PeerRepository) CreatePeer(ctx context.Context, userID int64, name string, port uint32, address string) (int64, error) {
	query := `INSERT INTO peers (user_id, name, port, address) VALUES ($1, $2, $3, $4) RETURNING id`
	var peerID int64
	row := r.db.QueryRow(query, userID, name, port, address)
	if err := row.Scan(&peerID); err != nil {
		return -1, err
	}
	return peerID, nil
}
