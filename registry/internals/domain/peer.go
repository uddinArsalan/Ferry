package domain

type Peer struct {
	ID      int64
	PeerID  string
	UserID  int64
	Name    string
	Port    uint32
	Address string
}
