package peer_service

import (
	"context"

	genpeer "github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	ErrPeerNotFound = status.Error(codes.NotFound, "peer not found")
)

type PeerService struct {
	peerRepo *repository.PeerRepository
	genpeer.UnimplementedPeerServiceServer
}

func NewPeerService(peerRepo *repository.PeerRepository) *PeerService {
	return &PeerService{
		peerRepo: peerRepo,
	}
}

// register a new peer and return the peer id
func (p *PeerService) RegisterPeer(ctx context.Context, req *genpeer.RegisterPeerRequest) (*genpeer.PeerID, error) {
	userID, ok := interceptor.GetUserIDFromContext(ctx)
	if !ok {
		return nil, interceptor.ErrUnauthenticated
	}
	peerID, err := p.peerRepo.CreatePeer(ctx, userID, req.PeerId, req.Name, req.Port, req.Address)
	if err != nil {
		return nil, err
	}
	return &genpeer.PeerID{
		Id: peerID,
	}, nil
}

// return peer details for a given peer id
func (p *PeerService) GetPeer(ctx context.Context, req *genpeer.GetPeerRequest) (*genpeer.Peer, error) {
	peer, err := p.peerRepo.GetPeerByPeerID(ctx, req.PeerId)
	if err != nil {
		return nil, err
	}
	if peer == nil {
		return nil, ErrPeerNotFound
	}
	return &genpeer.Peer{
		Id:      peer.ID,
		PeerId:  peer.PeerID,
		Name:    peer.Name,
		Port:    peer.Port,
		Address: peer.Address,
	}, nil
}

// return all peers belonging to user
func (p *PeerService) GetPeers(ctx context.Context, req *emptypb.Empty) (*genpeer.Peers, error) {
	userID, ok := interceptor.GetUserIDFromContext(ctx)
	if !ok {
		return nil, interceptor.ErrUnauthenticated
	}
	peers, err := p.peerRepo.GetPeersForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var peerList []*genpeer.Peer
	for _, peer := range peers {
		peerList = append(peerList, &genpeer.Peer{
			Id:      peer.ID,
			PeerId:  peer.PeerID,
			Name:    peer.Name,
			Port:    peer.Port,
			Address: peer.Address,
		})
	}
	return &genpeer.Peers{
		Peers: peerList,
	}, nil
}
