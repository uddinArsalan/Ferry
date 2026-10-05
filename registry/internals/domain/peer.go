package domain

type Peer struct {
	ID      int64
	UserID  int64
	Name    string
	Port    uint32
	Address string
}
