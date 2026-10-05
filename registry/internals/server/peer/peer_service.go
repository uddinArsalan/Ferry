package peer_service

import (
	"context"

	genpeer "github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	"google.golang.org/protobuf/types/known/emptypb"
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
	peerID, err := p.peerRepo.CreatePeer(ctx, userID, req.Name, req.Port, req.Address)
	if err != nil {
		return nil, err
	}
	return &genpeer.PeerID{
		Id: peerID,
	}, nil
}

// return peer details for a given peer id
func (p *PeerService) GetPeer(ctx context.Context, req *genpeer.PeerID) (*genpeer.Peer, error) {
	return nil, nil
}

// return all peers belonging to user
func (p *PeerService) GetPeers(ctx context.Context, req *emptypb.Empty) (*genpeer.Peers, error) {
	return nil, nil
}
