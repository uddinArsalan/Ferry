package tcp

import (
	"fmt"
	"log"
	"net"

	"github.com/uddinArsalan/ferry/peers"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	blockChan  chan struct{}
	pm         *peers.PeerManager
}

func NewServer(listenAddr string, pm *peers.PeerManager) *Server {
	return &Server{
		listenAddr: listenAddr,
		blockChan:  make(chan struct{}),
		pm:         pm,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln
	go s.AcceptConn()
	<-s.blockChan

	return nil
}

func (s *Server) AcceptConn() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			continue
		}
		log.Printf("Client connected %v", conn.RemoteAddr().String())
		if remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			remoteIP := remoteAddr.IP
			remotePort := remoteAddr.Port
			peerId, err := s.pm.AddPeer(remoteIP.String(), remotePort, conn)
			if err == nil {
				s.pm.SetPeerId(peerId)
			}
		}

		go s.ReadFromConn(conn)
	}
}

func (s Server) ReadFromConn(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			continue
		}
		msg := string(buf[:n])
		log.Printf("Recieved message from %v msg = %v", conn.RemoteAddr(), msg)
	}
}

func (s Server) ConnectToPeers() {
	for {
		for _, peerId := range s.pm.GetPeersToConnect() {
			peer, ok := s.pm.GetPeerConn(peerId)
			if !ok {
				continue
			}
			_, err := net.Dial("tcp", fmt.Sprintf("%v:%v", peer.Address, peer.Port))
			if err != nil {
				continue
			}
		}
	}

}
