package mapsync

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/uddinArsalan/ferry-proto/group"
	"github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry/cli"
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

// after a defined interval update local configurations
func (ms MapSyncWorker) Run() {
	go func() {
		defer ms.ticker.Stop()
		if err := ms.refresh(); err != nil {
			ms.l.Printf("error refreshing config: %v", err)
		}
		for {
			select {
			case <-ms.ctx.Done():
				return
			case <-ms.ticker.C:
				err := ms.refresh()
				if err != nil {
					ms.l.Printf("error refreshing config: %v", err)
				}
			}
		}
	}()
}

func (ms MapSyncWorker) refresh() error {
	currPeerID, err := ms.pm.GetPeerId()
	if err != nil || currPeerID == "" {
		return errors.New("unauthenticated or some problem with config")
	}
	authContext, err := cli.AuthContext()
	if err != nil {
		return errors.New("Unauthenticated request")
	}
	g, err := ms.groupClient.GetPeerSyncSnapshot(ms.ctx, &peer.GetPeerRequest{
		PeerId: currPeerID,
	}, authContext)
	if err != nil {
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
	localPeerID, err := ms.pm.GetPeerId()
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
			localDir := ""
			if peer.PeerId == localPeerID {
				localDir = localDirs[group.GroupId]
			}
			localGroups[group.GroupId].Peers[peer.PeerId] = types.PeerLocal{
				Peer: types.Peer{
					ID:        peer.PeerId,
					Name:      peer.Name,
					Address:   peer.Address,
					Port:      peer.Port,
					LastSeen:  peer.LastSeen.AsTime(),
					Connected: false,
				},
				DirPath: localDir,
			}
		}
	}
	ms.pm.UpdateGroups(localGroups)
	return nil
}
