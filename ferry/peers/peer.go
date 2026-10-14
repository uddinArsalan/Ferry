package peers

import (
	// "encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/uddinArsalan/ferry/keyring"
	"github.com/uddinArsalan/ferry/types"
)

// we might need mutexes

var ErrGrpNotFound = errors.New("no group found with provided group id")
var ErrPeerNotFound = errors.New("no peer found with provided peer id")

type PeerManager struct {
	mu            sync.RWMutex
	currPeerId    string
	peers         map[string]*types.Peer         // peerID -> peer detail
	groups        map[string]*types.SyncGroup    // syncID/grpId -> group
	peersGroupMap map[string]map[string]struct{} // peerId -> set of syncIds/groupIds
	connMap       map[string]map[string]net.Conn // map of peer id to its map of connections with other peers
}

func NewPeerManager() *PeerManager {
	currPeerId, err := keyring.GetPeerId()
	if err != nil {
		currPeerId = ""
	}
	return &PeerManager{
		currPeerId:    currPeerId,
		peers:         make(map[string]*types.Peer),
		peersGroupMap: make(map[string]map[string]struct{}),
		groups:        make(map[string]*types.SyncGroup),
		connMap:       make(map[string]map[string]net.Conn),
	}
}

func (p *PeerManager) SetPeerId(peerId string) {
	p.currPeerId = peerId
}

func (p *PeerManager) GetPeerId() string {
	return p.currPeerId
}

func (p *PeerManager) AddPeer(peerId, name, address string, port uint32) {
	p.SetPeerId(peerId)
	peer := types.Peer{
		ID:        peerId,
		Name:      name,
		Address:   address,
		Port:      port,
		LastSeen:  time.Now(),
		Connected: false,
	}
	p.mu.Lock()
	p.peers[peerId] = &peer
	p.mu.Unlock()
}

func (p *PeerManager) GetPeerDetail(peerId string) (*types.Peer, bool) {
	peer, ok := p.peers[peerId]
	if !ok {
		return nil, false
	}
	return peer, true
}

func (p *PeerManager) GetPeerConnections(peerId string) (*map[string]net.Conn, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	connections, ok := p.connMap[peerId]
	return &connections, ok
}

func (p *PeerManager) GetGroup(grpId string) (*types.SyncGroup, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	group, ok := p.groups[grpId]
	if !ok {
		return nil, false
	}
	return group, true
}

func (p *PeerManager) GetGroupPeers(grpId string) (map[string]types.PeerLocal, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	group, ok := p.GetGroup(grpId)
	if !ok {
		return nil, false
	}
	return group.Peers, true
}

func (p *PeerManager) AddPeerToGroup(peer types.Peer, grpId, localDir string) error {
	// _, ok := p.GetPeerDetail(peerId)
	// if !ok {
	// 	return ErrPeerNotFound
	// }
	group, ok := p.GetGroup(grpId)
	if !ok {
		return ErrGrpNotFound
	}
	p.mu.Lock()
	group.Peers[peer.ID] = types.PeerLocal{
		Peer: types.Peer{
			ID:        peer.ID,
			Name:      peer.Name,
			Address:   peer.Address,
			Port:      peer.Port,
			LastSeen:  peer.LastSeen,
			Connected: false,
		},
		DirPath: localDir,
	}
	_, ok = p.peersGroupMap[peer.ID]
	if !ok {
		p.peersGroupMap[peer.ID] = map[string]struct{}{}
	}
	p.peersGroupMap[peer.ID][grpId] = struct{}{}
	p.mu.Unlock()
	return nil
}

func (p *PeerManager) AddGroup(grpId, name string) {
	p.mu.Lock()
	p.groups[grpId] = &types.SyncGroup{
		GroupID: grpId,
		Name:    name,
		Peers:   map[string]types.PeerLocal{},
	}
	p.mu.Unlock()
}

func (p *PeerManager) RemovePeerFromAllGroups(peerId string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for grpId := range p.groups {
		_, ok := p.groups[grpId].Peers[peerId]
		if ok {
			delete(p.groups[grpId].Peers, peerId)
		}
		p.peersGroupMap[peerId] = map[string]struct{}{}
	}
}

func (p *PeerManager) RemovePeerFromGroup(peerId string, grpId string) error {
	// _, ok := p.GetPeerDetail(peerId)
	// if !ok {
	// 	return ErrPeerNotFound
	// }
	_, ok := p.GetGroup(grpId)
	if !ok {
		return ErrGrpNotFound
	}
	p.mu.Lock()
	delete(p.peersGroupMap[peerId], grpId)
	delete(p.groups[grpId].Peers, peerId)
	p.mu.Unlock()
	return nil
}

func (p *PeerManager) GetPeersToConnect() []types.PeerLocal {
	peersArr := make([]types.PeerLocal, 0)
	currPeerId := p.GetPeerId()
	p.mu.RLock()
	defer p.mu.RUnlock()
	// provides all groups for which peer is a member
	for grpId := range p.peersGroupMap[currPeerId] {
		peers, ok := p.GetGroupPeers(grpId)
		if !ok {
			return nil
		}
		for peerId := range peers {
			peersArr = append(peersArr, peers[peerId])
		}

	}
	return peersArr
}

// func (p *PeerManager) SendToAllPeersOfGroup(eventChan chan types.Event) {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	currPeerId := p.GetPeerId()
// 	for event := range eventChan {
// 		// need something else here
// 		data, err := json.Marshal(event)
// 		if err != nil {
// 			continue
// 		}
// 		for _, peerId := range p.GetPeersToConnect() {
// 			p.connMap[currPeerId][peerId].Write(data)
// 		}
// 	}
// }

func (p *PeerManager) GetPeersConn(peerId string) net.Conn {
	currPeerId := p.GetPeerId()
	return p.connMap[currPeerId][peerId]
}

// when a peer init we have to define a peer id local, and it will start listening for conn
// if a peer gets added to a group , g1 how a curr peer knows there is some new peer and we have
// to make a new conn with it, so registry helps here , we have to basically for each group get peers
// and prepare a local map and all
// second a peer init is waiting for a conn , so other peer has to either dial or this peer to all
// other peers what we can do the above each group a curr peer is part of get the member per group
// it will help in generating the new peers curr peer to conn to once conn is established
// so it has to be done in also in background and may be in init only after server start
// we store in conn map
// and then for sync for a grp id for curr peer sync all members by writing data for now say. peer id or some thing
