package mapsync

import (
	"context"
	"log"
	"time"

	"github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry/config"
	"github.com/uddinArsalan/ferry/peers"
	"github.com/uddinArsalan/ferry/types"
)

type MapSyncWorker struct {
	ctx         context.Context
	l           *log.Logger
	pm          *peers.PeerManager
	ticker      *time.Ticker
	groupClient group.GroupServiceClient
}

func NewMapSyncer(
	ctx context.Context,
	l *log.Logger,
	pm *peers.PeerManager,
	groupClient group.GroupServiceClient,
	interval int,
) MapSyncWorker {
	return MapSyncWorker{
		ctx:         ctx,
		l:           l,
		pm:          pm,
		groupClient: groupClient,
		ticker:      time.NewTicker(time.Duration(interval) * time.Second),
	}
}

var MAX_RETRY = 3

// after a defined interval update local configurations
func (ms MapSyncWorker) Run() {
	go func() {
		defer ms.ticker.Stop()
		errCnt := 0
		for {
			select {
			case <-ms.ctx.Done():
				return
			case <-ms.ticker.C:
				err := ms.refresh()
				if err != nil {
					errCnt++
					log.Printf("error refreshing config %v", err)
					if errCnt > MAX_RETRY {
						return
					}
				}
			}
		}
	}()
}

func (ms MapSyncWorker) refresh() error {
	currPeerID, err := ms.pm.GetPeerId()
	if err != nil || currPeerID == "" {
		ms.l.Fatalf("unauthenticated or some problem with config")
	}
	g, err := ms.groupClient.GetPeerSyncSnapshot(ms.ctx, &peer.GetPeerRequest{
		PeerId: currPeerID,
	})
	if err != nil {
		ms.l.Printf("Error: %v", err.Error())
		return err
	}
	// update peer -> groups config
	ms.updatePeerGroup(currPeerID, g)

	// update groups
	err = ms.updateGroups(g)
	if err != nil {
		return err
	}
	return nil
}

func (ms MapSyncWorker) updatePeerGroup(peerID string, groups *group.SnapshotResponse) {
	peersGroupMap := make(map[string]map[string]struct{})
	for _, group := range groups.Groups {
		_, ok := peersGroupMap[peerID]
		if !ok {
			peersGroupMap[peerID] = map[string]struct{}{}
		}
		peersGroupMap[peerID][group.GroupId] = struct{}{}
	}
	ms.pm.UpdatePeerGroup(peersGroupMap)
}

func (ms MapSyncWorker) updateGroups(groups *group.SnapshotResponse) error {
	localDirs, err := config.GetLocalDirs()
	if err != nil {
		return err
	}
	localGroups := make(map[string]*types.SyncGroup)
	for _, group := range groups.Groups {
		localGroups[group.GroupId] = &types.SyncGroup{
			GroupID: group.GroupId,
			Name:    group.Name,
			Peers:   make(map[string]types.PeerLocal),
		}
		for _, peer := range group.Peer {
			localGroups[group.GroupId].Peers[peer.PeerId] = types.PeerLocal{
				Peer: types.Peer{
					ID:        peer.PeerId,
					Name:      peer.Name,
					Address:   peer.Address,
					Port:      peer.Port,
					LastSeen:  peer.LastSeen.AsTime(),
					Connected: false,
				},
				DirPath: localDirs[group.GroupId],
			}
		}
	}
	ms.pm.UpdateGroups(localGroups)
	return nil
}
