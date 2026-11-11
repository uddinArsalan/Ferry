package domain

import "time"

type Group struct {
	ID        int64
	GroupID   string
	Name      string
	UserID    int64
	CreatedAt time.Time
}

type GroupWithPeers struct {
	Id        int64
	GroupId   string
	Name      string
	CreatedAt time.Time
	Peers     []Peer
}
