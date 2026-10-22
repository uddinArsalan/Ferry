package group_service

import (
	"context"
	"log"

	gengroup "github.com/uddinArsalan/ferry-proto/group"
	genpeer "github.com/uddinArsalan/ferry-proto/peer"
	"github.com/uddinArsalan/ferry-registry/interceptor"
	"github.com/uddinArsalan/ferry-registry/internals/domain"
	"github.com/uddinArsalan/ferry-registry/internals/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// var ErrGroupNotFound = status.Error(codes.NotFound, "Group not found")

type GroupServer struct {
	groupRepo *repository.GroupRepository
	gengroup.UnimplementedGroupServiceServer
}

func NewGroupServer(groupRepo *repository.GroupRepository) *GroupServer {
	return &GroupServer{
		groupRepo: groupRepo,
	}
}

func (g *GroupServer) CreateGroup(ctx context.Context, req *gengroup.CreateGroupRequest) (*gengroup.GroupID, error) {
	groupID, err := g.groupRepo.CreateGroup(ctx, req.GroupId, req.Name)
	if err != nil {
		return nil, err
	}
	return &gengroup.GroupID{
		Id: groupID,
	}, nil
}

func (g *GroupServer) GetGroupsForPeer(ctx context.Context, req *genpeer.GetPeerRequest) (*gengroup.Groups, error) {
	groups, err := g.groupRepo.GetGroupsForPeer(ctx, req.PeerId)
	if err != nil {
		log.Printf("Error retrieving groups for peer %d: %v", req.PeerId, err)
		return nil, err
	}
	var groupList []*gengroup.Group
	for _, group := range groups {
		groupList = append(groupList, &gengroup.Group{
			Id:        group.ID,
			GroupId:   group.GroupID,
			Name:      group.Name,
			UserId:    group.UserID,
			CreatedAt: timestamppb.New(group.CreatedAt),
		})
	}
	return &gengroup.Groups{
		Groups: groupList,
	}, nil
}

func (g *GroupServer) GetGroups(ctx context.Context, void *emptypb.Empty) (*gengroup.Groups, error) {
	userID, ok := interceptor.GetUserIDFromContext(ctx)
	if !ok {
		return nil, interceptor.ErrUnauthenticated
	}
	groups, err := g.groupRepo.GetGroupsForUser(ctx, userID)
	if err != nil {
		log.Printf("Error retrieving groups for user %d: %v", userID, err)
		return nil, err
	}
	var groupList []*gengroup.Group
	for _, group := range groups {
		groupList = append(groupList, &gengroup.Group{
			Id:        group.ID,
			GroupId:   group.GroupID,
			Name:      group.Name,
			UserId:    group.UserID,
			CreatedAt: timestamppb.New(group.CreatedAt),
		})
	}
	return &gengroup.Groups{
		Groups: groupList,
	}, nil
}

func (g *GroupServer) GetGroup(ctx context.Context, req *gengroup.GetGroupRequest) (*gengroup.Group, error) {
	group, err := g.groupRepo.GetGroupByGrpID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, status.Error(codes.NotFound, "Group not found")
	}
	return &gengroup.Group{
		Id:        group.ID,
		GroupId:   group.GroupID,
		Name:      group.Name,
		UserId:    group.UserID,
		CreatedAt: timestamppb.New(group.CreatedAt),
	}, nil
}

func (g *GroupServer) CreateGroupPeerMembership(
	ctx context.Context,
	req *gengroup.CreateGroupPeerMembershipRequest,
) (*emptypb.Empty, error) {
	if err := g.groupRepo.CreateGroupPeerMembership(ctx,
		req.PeerId,
		req.GroupId,
		domain.PeerRole(req.Role)); err != nil {
		return nil, err
	}
	return nil, nil
}

func (g *GroupServer) GetGroupPeerMembership(
	ctx context.Context,
	req *gengroup.GetGroupPeerMembershipRequest,
) (*gengroup.GetGroupPeerMembershipResponse, error) {
	mem, err := g.groupRepo.GetGroupPeerMembership(ctx,
		req.PeerId,
		req.GroupId)
	if err != nil {
		return nil, err
	}
	return &gengroup.GetGroupPeerMembershipResponse{
		PeerId:   mem.PeerID,
		GroupId:  mem.GroupID,
		Role:     gengroup.PeerRole(mem.Role),
		JoinedAt: mem.JoinedAt,
	}, nil
}

// create join request (invite/join) for a particular group from
// sender peer to a recipient peer
func (g *GroupServer) CreateJoinRequest(
	ctx context.Context,
	req *gengroup.CreateJoin,
) (*emptypb.Empty, error) {
	err := g.groupRepo.CreateJoinRequest(ctx,
		req.GroupId,
		req.SenderPeerId,
		req.RecipientPeerId,
		domain.JoinType(req.Type),
	)
	return nil, err
}

// get join/membership request (invite/join)
func (g *GroupServer) GetJoinRequest(
	ctx context.Context,
	req *gengroup.GetJoinReq,
) (*gengroup.GetJoinRequestResponse, error) {
	j, err := g.groupRepo.GetJoinRequest(ctx, req.RequestId)
	if err != nil {
		return nil, err
	}
	return &gengroup.GetJoinRequestResponse{
		RequestId:         j.RequestID,
		GroupId:           j.GroupID,
		GroupName:         j.GroupName,
		Type:              gengroup.JoinType(j.JoinType),
		SenderPeerId:      j.SenderPeerID,
		SenderPeerName:    j.SenderPeerName,
		RecipientPeerId:   j.RecipientPeerID,
		RecipientPeerName: j.RecipientPeerName,
		Status:            gengroup.INVITE_STATUS(j.JoinStatus),
	}, nil
}

// update status
func (g *GroupServer) UpdateJoinRequestStatus(
	ctx context.Context,
	req *gengroup.UpdateJoin,
) (*emptypb.Empty, error) {
	if err := g.groupRepo.UpdateJoinRequestStatus(
		ctx,
		req.RequestId,
		domain.JoinStatus(req.Status),
	); err != nil {
		return nil, err
	}
	return nil, nil
}

// get all pending invites for a peer (accross all groups from each group owner)
func (g *GroupServer) GetAllPendingInvites(
	ctx context.Context,
	req *gengroup.GetAllPendingRequest,
) (*gengroup.PendingInviteResponse, error) {
	pendingInvites, err := g.groupRepo.GetAllPendingInvites(ctx, req.PeerId)
	if err != nil {
		return nil, err
	}
	var pendingInviteRequests []*gengroup.GetInviteRequestResponse
	for _, request := range pendingInvites {
		pendingInviteRequests = append(pendingInviteRequests,
			&gengroup.GetInviteRequestResponse{
				RequestId:         request.RequestID,
				GroupId:           request.GroupID,
				GroupName:         request.GroupName,
				Type:              gengroup.JoinType(request.JoinType),
				SenderPeerId:      request.SenderPeerID,
				SenderPeerName:    request.SenderPeerName,
				RecipientPeerId:   request.RecipientPeerID,
				RecipientPeerName: request.RecipientPeerName,
				Status:            gengroup.INVITE_STATUS(request.JoinStatus),
			})
	}
	return &gengroup.PendingInviteResponse{
		PendingInviteRequests: pendingInviteRequests,
	}, nil
}

func (g *GroupServer) GetAllPendingJoinRequestsBySender(
	ctx context.Context,
	req *gengroup.GetAllPendingRequest,
) (*gengroup.PendingJoinResponse, error) {
	pendingJoins, err := g.groupRepo.GetAllPendingJoinRequestsBySender(ctx, req.PeerId)
	if err != nil {
		return nil, err
	}
	var pendingJoinRequests []*gengroup.GetJoinRequestResponse
	for _, request := range pendingJoins {
		pendingJoinRequests = append(pendingJoinRequests,
			&gengroup.GetJoinRequestResponse{
				RequestId:         request.RequestID,
				GroupId:           request.GroupID,
				GroupName:         request.GroupName,
				Type:              gengroup.JoinType(request.JoinType),
				SenderPeerId:      request.SenderPeerID,
				SenderPeerName:    request.SenderPeerName,
				RecipientPeerId:   request.RecipientPeerID,
				RecipientPeerName: request.RecipientPeerName,
				Status:            gengroup.INVITE_STATUS(request.JoinStatus),
			})
	}
	return &gengroup.PendingJoinResponse{
		PendingJoinRequests: pendingJoinRequests,
	}, nil
}

func (g *GroupServer) GetAllPendingJoinRequestsForOwner(
	ctx context.Context,
	req *gengroup.GetAllPendingRequest,
) (*gengroup.PendingJoinResponse, error) {
	pendingJoins, err := g.groupRepo.GetAllPendingJoinRequestsForOwner(ctx, req.PeerId)
	if err != nil {
		return nil, err
	}
	var pendingJoinRequests []*gengroup.GetJoinRequestResponse
	for _, request := range pendingJoins {
		pendingJoinRequests = append(pendingJoinRequests,
			&gengroup.GetJoinRequestResponse{
				RequestId:         request.RequestID,
				GroupId:           request.GroupID,
				GroupName:         request.GroupName,
				Type:              gengroup.JoinType(request.JoinType),
				SenderPeerId:      request.SenderPeerID,
				SenderPeerName:    request.SenderPeerName,
				RecipientPeerId:   request.RecipientPeerID,
				RecipientPeerName: request.RecipientPeerName,
				Status:            gengroup.INVITE_STATUS(request.JoinStatus),
			})
	}
	return &gengroup.PendingJoinResponse{
		PendingJoinRequests: pendingJoinRequests,
	}, nil
}

func (g *GroupServer) GetGroupPendingInvite(
	ctx context.Context,
	req *gengroup.GetGroupPendingRequest,
) (*gengroup.GetInviteRequestResponse, error) {
	pendingGroupRequest, err := g.groupRepo.GetGroupPendingInvite(
		ctx,
		req.PeerId,
		req.GroupId)
	if err != nil {
		return nil, err
	}
	return &gengroup.GetInviteRequestResponse{
		RequestId:         pendingGroupRequest.RequestID,
		GroupId:           pendingGroupRequest.GroupID,
		GroupName:         pendingGroupRequest.GroupName,
		SenderPeerId:      pendingGroupRequest.SenderPeerID,
		SenderPeerName:    pendingGroupRequest.SenderPeerName,
		RecipientPeerId:   pendingGroupRequest.RecipientPeerID,
		RecipientPeerName: pendingGroupRequest.RecipientPeerName,
		Type:              gengroup.JoinType(pendingGroupRequest.JoinType),
		Status:            gengroup.INVITE_STATUS(pendingGroupRequest.JoinStatus),
	}, nil
}

func (g *GroupServer) GetGroupPendingJoinRequestsBySender(
	ctx context.Context,
	req *gengroup.GetGroupPendingRequest,
) (*gengroup.GetJoinRequestResponse, error) {
	pendingGroupRequest, err := g.groupRepo.GetGroupPendingJoinRequestsBySender(
		ctx,
		req.PeerId,
		req.GroupId)
	if err != nil {
		return nil, err
	}
	return &gengroup.GetJoinRequestResponse{
		RequestId:         pendingGroupRequest.RequestID,
		GroupId:           pendingGroupRequest.GroupID,
		GroupName:         pendingGroupRequest.GroupName,
		SenderPeerId:      pendingGroupRequest.SenderPeerID,
		SenderPeerName:    pendingGroupRequest.SenderPeerName,
		RecipientPeerId:   pendingGroupRequest.RecipientPeerID,
		RecipientPeerName: pendingGroupRequest.RecipientPeerName,
		Type:              gengroup.JoinType(pendingGroupRequest.JoinType),
		Status:            gengroup.INVITE_STATUS(pendingGroupRequest.JoinStatus),
	}, nil
}

func (g *GroupServer) GetGroupPendingJoinRequestsForOwner(
	ctx context.Context,
	req *gengroup.GetGroupPendingRequest,
) (*gengroup.PendingJoinResponse, error) {
	pendingJoins, err := g.groupRepo.GetGroupPendingJoinRequestsForOwner(
		ctx,
		req.PeerId,
		req.GroupId)
	if err != nil {
		return nil, err
	}
	var pendingJoinRequests []*gengroup.GetJoinRequestResponse
	for _, request := range pendingJoins {
		pendingJoinRequests = append(pendingJoinRequests,
			&gengroup.GetJoinRequestResponse{
				RequestId:         request.RequestID,
				GroupId:           request.GroupID,
				GroupName:         request.GroupName,
				Type:              gengroup.JoinType(request.JoinType),
				SenderPeerId:      request.SenderPeerID,
				SenderPeerName:    request.SenderPeerName,
				RecipientPeerId:   request.RecipientPeerID,
				RecipientPeerName: request.RecipientPeerName,
				Status:            gengroup.INVITE_STATUS(request.JoinStatus),
			})
	}
	return &gengroup.PendingJoinResponse{
		PendingJoinRequests: pendingJoinRequests,
	}, nil
}
