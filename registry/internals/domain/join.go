package domain

import "time"

type JoinType int

const (
	JOIN_REQUEST JoinType = iota
	INVITE
)

var joinTypeMap = map[JoinType]string{
	JOIN_REQUEST: "JOIN_REQUEST",
	INVITE:       "INVITE",
}

func (j JoinType) String() string {
	return joinTypeMap[j]
}

type JoinStatus int

const (
	PENDING JoinStatus = iota
	ACCEPTED
	REJECTED
	CANCELLED
)

var joinStatusMap = map[JoinStatus]string{
	PENDING:   "PENDING",
	ACCEPTED:  "ACCEPTED",
	REJECTED:  "REJECTED",
	CANCELLED: "CANCELLED",
}

func (j JoinStatus) String() string {
	return joinStatusMap[j]
}

type JoinRequest struct {
	RequestID         int64
	GroupID           string
	GroupName         string
	SenderPeerID      string
	SenderPeerName    string
	RecipientPeerID   string
	RecipientPeerName string
	JoinType          JoinType
	JoinStatus        JoinStatus
	CreatedAt         time.Time
}
