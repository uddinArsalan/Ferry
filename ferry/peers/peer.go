package peers

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

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
	return &PeerManager{
		peers:         make(map[string]*types.Peer),
		peersGroupMap: make(map[string]map[string]struct{}),
		groups:        make(map[string]*types.SyncGroup),
	}
}

func (p *PeerManager) SetPeerId(peerId string) {
	p.currPeerId = peerId
}

func (p *PeerManager) GetPeerId() string {
	return p.currPeerId
}

func (p *PeerManager) AddPeer(peerId, address string, port uint32) {
	p.SetPeerId(peerId)
	peer := types.Peer{
		ID:        peerId,
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

func (p *PeerManager) AddPeerToGroup(peerId, grpId, localDir string) error {
	_, ok := p.GetPeerDetail(peerId)
	if !ok {
		return ErrPeerNotFound
	}
	group, ok := p.GetGroup(grpId)
	if !ok {
		return ErrGrpNotFound
	}
	p.mu.Lock()
	group.Peers[peerId] = types.PeerLocal{
		PeerID:  peerId,
		DirPath: localDir,
	}
	_, ok = p.peersGroupMap[peerId]
	if !ok {
		p.peersGroupMap[peerId] = map[string]struct{}{}
	}
	p.peersGroupMap[peerId][grpId] = struct{}{}
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
	_, ok := p.GetPeerDetail(peerId)
	if !ok {
		return ErrPeerNotFound
	}
	_, ok = p.GetGroup(grpId)
	if !ok {
		return ErrGrpNotFound
	}
	p.mu.Lock()
	delete(p.peersGroupMap[peerId], grpId)
	delete(p.groups[grpId].Peers, peerId)
	p.mu.Unlock()
	return nil
}

func (p *PeerManager) GetPeersToConnect() []string {
	peerIds := make([]string, 0)
	currPeerId := p.GetPeerId()
	p.mu.RLock()
	defer p.mu.RUnlock()
	// provides all groups for which peer is a member
	for grpId := range p.peersGroupMap[currPeerId] {
		peers, ok := p.GetGroupPeers(grpId)
		if !ok {
			return []string{}
		}
		for peerId := range peers {
			peerIds = append(peerIds, peerId)
		}
	}
	return peerIds
}

func (p *PeerManager) SendToAllPeersOfGroup(eventChan chan types.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	currPeerId := p.GetPeerId()
	for event := range eventChan {
		// need something else here
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		for _, peerId := range p.GetPeersToConnect() {
			p.connMap[currPeerId][peerId].Write(data)
		}
	}
}

func (p *PeerManager) GetPeersConn(peerId string) net.Conn {
	currPeerId := p.GetPeerId()
	return p.connMap[currPeerId][peerId]
}
