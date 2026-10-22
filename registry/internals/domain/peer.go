package domain

import (
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Peer struct {
	ID      int64
	PeerID  string
	UserID  int64
	Name    string
	Port    uint32
	Address string
}

type PeerRole int

const (
	OWNER PeerRole = iota
	ADMIN
	MEMBER
)

var roleMap = map[PeerRole]string{
	OWNER:  "OWNER",
	ADMIN:  "ADMIN",
	MEMBER: "MEMBER",
}

func (r PeerRole) String() string {
	return roleMap[r]
}

type PeerGroupMembership struct {
	GroupID  int64
	PeerID   int64
	Role     PeerRole
	JoinedAt *timestamppb.Timestamp
}
