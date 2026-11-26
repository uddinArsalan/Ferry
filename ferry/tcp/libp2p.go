package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/types"
)

type Node struct {
	ctx       context.Context
	node      host.Host
	pm        *peers.PeerManager
	eventChan chan types.Event
	ticker    *time.Ticker
	log       *log.Logger
}

var (
	syncID = protocol.ID("/ferry/sync/1.0.0")
)

func NewNode(ctx context.Context, privKey crypto.PrivKey, pm *peers.PeerManager, eventChan chan types.Event, interval int, log *log.Logger) (*Node, error) {
	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Minute),
	)

	node, err := libp2p.New(
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Identity(privKey),
		libp2p.ConnectionManager(connmgr),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
	)

	if err != nil {
		return nil, err
	}

	node.SetStreamHandler(syncID, handleStream)
	return &Node{ctx: ctx, node: node, pm: pm, eventChan: eventChan, ticker: time.NewTicker(time.Duration(interval) * time.Second), log: log}, nil
}

func (n *Node) Close() error {
	return n.node.Close()
}

func (n *Node) GetAddr() string {
	if len(n.node.Addrs()) > 1 {
		return n.node.Addrs()[1].String()
	}
	return n.node.Addrs()[0].String()
}

func (n *Node) GetAddrsAndPort() []string {
	addrs := strings.Split(n.GetAddr(), "/")
	return []string{addrs[2], addrs[4]}
}

func (n *Node) GetID() string {
	return n.node.ID().String()
}

func (n *Node) GetConnectedPeers() chan peer.ID {
	peerIDChan := make(chan peer.ID)
	n.node.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			peerIDChan <- c.RemotePeer()
		},
	})
	return peerIDChan
}

// after a interval try to connect to other peers (curr peer has contact with)
func (n *Node) Start() {
	go func() {
		for {
			select {
			case <-n.ctx.Done():
				return
			case event := <-n.eventChan:
				n.WriteData(event.GroupID, Message{
					Content: fmt.Sprintf("%s %s", event.Op, event.Path),
				})
			}
		}
	}()

	go func() {
		defer n.ticker.Stop()
		for {
			select {
			case <-n.ctx.Done():
				log.Println("Node stopped")
				return
			case <-n.ticker.C:
				n.ConnectToPeers(n.pm.GetPeersToConnect())
			}
		}
	}()
}

func (n *Node) ConnectToPeers(peersToConnect []types.PeerLocal) {
	for _, p := range peersToConnect {
		_, err := n.pm.GetPeersConn(p.Peer.ID)
		if err != nil {
			n.log.Printf("connection already exists with this peer")
			continue
		}
		peerAddr := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", p.Peer.Address, p.Peer.Port, p.Peer.ID)
		peerInfo, err := peer.AddrInfoFromString(peerAddr)
		if err != nil {
			n.log.Printf("Error creating peer info: %v", err)
			continue
		}
		n.node.Peerstore().AddAddrs(
			peerInfo.ID,
			peerInfo.Addrs,
			time.Hour,
		)

		s, err := n.node.NewStream(context.Background(), peerInfo.ID, syncID)
		if err != nil {
			n.log.Printf("Error connecting to peer: %v", err)
			continue
		}
		log.Printf("Established connection to %v", p.Peer.Name)

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		n.pm.UpdatePeerConn(p.Peer.ID, rw)

		go readData(rw)

	}
}

func handleStream(s network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
}

func readData(rw *bufio.ReadWriter) {
	buf := make([]byte, 1024)
	for {
		n, err := rw.Read(buf)
		if err != nil {
			log.Printf("error reading data %v", err)
			return
		}
		log.Printf("Data readed %s", buf[:n])
	}

}

type Message struct {
	Content string
}

// this will write data to every peer of a group id
func (n *Node) WriteData(groupID string, msg Message) {
	peerMap, ok := n.pm.GetGroupPeers(groupID)
	if !ok {
		log.Println("No peers found to share data ")
		return
	}
	for peerID := range peerMap {
		fmt.Printf("sending message to peer %v: %+v", peerID, msg)
		rw, err := n.pm.GetPeersConn(peerID)
		if err != nil {
			log.Printf("error getting connection details %v", err)
			continue
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("invalid message format or error parsing %v", err)
			continue
		}
		if _, err := rw.Write(data); err != nil {
			log.Println("write error:", err)
			continue
		}

		if err := rw.Flush(); err != nil {
			log.Println("flush error:", err)
			continue
		}

		fmt.Println("message sent successfully")
	}
}
