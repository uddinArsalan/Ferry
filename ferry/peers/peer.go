package peers

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/uddinArsalan/ferry/types"
	"github.com/uddinArsalan/ferry/utils"
)

// we might need mutexes

var ErrGrpNotFound = errors.New("no group found with provided group id")
var ErrPeerNotFound = errors.New("no peer found with provided peer id")

type PeerManager struct {
	mu            sync.RWMutex
	currPeerId    string
	peers         map[string]*types.PeerConn     // peerID -> connection/runtime info
	groups        map[string]*types.SyncGroup    // syncID/grpId -> group
	peersGroupMap map[string]map[string]struct{} // peerId -> set of syncIds/groupIds
}

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers:         make(map[string]*types.PeerConn),
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

func (p *PeerManager) AddPeer(address string, port int, conn net.Conn) (string, error) {
	peerId, err := utils.NewId()
	if err != nil {
		return "", err
	}
	peerCon := types.PeerConn{
		Conn:      conn,
		LastSeen:  time.Now(),
		Connected: true,
		Peer: types.Peer{
			ID:      peerId,
			Address: address,
			Port:    port,
		},
	}
	p.mu.Lock()
	p.peers[peerId] = &peerCon
	p.mu.Unlock()
	return peerId, nil
}

func (p *PeerManager) GetPeerConn(peerId string) (*types.PeerConn, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peer, ok := p.peers[peerId]
	return peer, ok
}

func (p *PeerManager) GetGroup(grpId string) (*types.SyncGroup, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	group, ok := p.groups[grpId]
	return group, ok
}

func (p *PeerManager) GetGroupPeers(grpId string) (map[string]types.PeerLocal, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	group, ok := p.groups[grpId]
	return group.Peers, ok
}

func (p *PeerManager) AddPeerToGroup(peerId, grpId, localDir string) error {
	_, ok := p.GetPeerConn(peerId)
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
	p.peersGroupMap[peerId] = map[string]struct{}{}
	p.peersGroupMap[peerId][grpId] = struct{}{}
	p.mu.Unlock()
	return nil
}

func (p *PeerManager) GetPeersIDForGroup(grpId string) (map[string]types.PeerLocal, error) {
	group, ok := p.groups[grpId]
	if !ok {
		return nil, ErrGrpNotFound
	}
	return group.Peers, nil
}

func (p *PeerManager) AddGroup() (string, error) {
	grpId, err := utils.NewId()
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	p.groups[grpId] = &types.SyncGroup{
		SyncID: grpId,
		Peers:  map[string]types.PeerLocal{},
	}
	p.mu.Unlock()
	return grpId, nil
}

func (p *PeerManager) RemovePeerFromAllGroups(peerId string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for grpId := range p.groups {
		_, ok := p.groups[grpId].Peers[peerId]
		if ok {
			delete(p.groups[grpId].Peers, peerId)
		}
	}
}

func (p *PeerManager) RemovePeer(peerId string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	peer, ok := p.GetPeerConn(peerId)
	if !ok {
		return ErrPeerNotFound
	}
	peer.Conn.Close()
	p.RemovePeerFromAllGroups(peerId)
	return nil
}

func (p *PeerManager) GetPeersToConnect() []string {
	peerIds := make([]string, 0)
	currPeerId := p.GetPeerId()
	p.mu.RLock()
	defer p.mu.RUnlock()
	for grpId := range p.peersGroupMap[currPeerId] {
		for peerId := range p.groups[grpId].Peers {
			peerIds = append(peerIds, peerId)
		}
	}
	return peerIds
}

func (p *PeerManager) SendToAllPeersOfGroup(eventChan chan types.Event) {
	for event := range eventChan {
		// need something else here
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		for _, peerId := range p.GetPeersToConnect() {
			p.peers[peerId].Conn.Write(data)
		}
	}
}
