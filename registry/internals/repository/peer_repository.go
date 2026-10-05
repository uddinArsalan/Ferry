package repository

import (
	"context"
	"database/sql"

	"github.com/uddinArsalan/ferry-registry/internals/domain"
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
	row := r.db.QueryRowContext(ctx, query, userID, name, port, address)
	if err := row.Scan(&peerID); err != nil {
		return -1, err
	}
	return peerID, nil
}

func (r *PeerRepository) GetPeersForUser(ctx context.Context, userID int64) ([]*domain.Peer, error) {
	query := `SELECT id, user_id, name, port, address FROM peers WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*domain.Peer
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var peer domain.Peer
		if err := rows.Scan(&peer.ID, &peer.UserID, &peer.Name, &peer.Port, &peer.Address); err != nil {
			return nil, err
		}
		peers = append(peers, &peer)
	}

	return peers, nil
}

func (r *PeerRepository) GetPeerByID(ctx context.Context, peerID int64) (*domain.Peer, error) {
	query := `SELECT id, user_id, name, port, address FROM peers WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, peerID)
	var peer domain.Peer
	if err := row.Scan(&peer.ID, &peer.UserID, &peer.Name, &peer.Port, &peer.Address); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No peer found with the given ID
		}
		return nil, err
	}
	return &peer, nil
}
