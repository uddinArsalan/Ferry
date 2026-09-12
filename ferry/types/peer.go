package types

import (
	"net"
	"time"
)

type Peer struct {
	ID      string
	Address string
	Port    int
}

type PeerConn struct {
	Peer
	Conn      net.Conn
	LastSeen  time.Time
	Connected bool
}

// PeerLocal represents a peer's config *within* one sync group.
type PeerLocal struct {
	PeerID  string 
	DirPath string // local root dir for this group, on that peer's machine
}

// A sync group contains the peers that synchronize the same logical directory.
type SyncGroup struct {
	SyncID string
	Peers  map[string]PeerLocal
}
