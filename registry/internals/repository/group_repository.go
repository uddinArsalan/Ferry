package repository

import (
	"context"
	"database/sql"

	"github.com/uddinArsalan/ferry-registry/internals/domain"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) CreateGroup(ctx context.Context, localGroupID, name string) (int64, error) {
	query := `INSERT INTO groups (name,group_id) VALUES ($1,$2) RETURNING id`
	// this represent db table pk
	var groupID int64
	row := r.db.QueryRowContext(ctx, query, name, localGroupID)
	if err := row.Scan(&groupID); err != nil {
		return -1, err
	}
	return groupID, nil
}

func (r *GroupRepository) GetGroupsForUser(ctx context.Context, userID int64) ([]*domain.Group, error) {
	query := "SELECT id, name, user_id,group_id, created_at FROM groups WHERE user_id = $1"
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*domain.Group
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var group domain.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	return groups, nil
}

func (r *GroupRepository) GetGroupByGrpID(ctx context.Context, localGroupID string) (*domain.Group, error) {
	query := "SELECT id, name, user_id,group_id, created_at FROM groups WHERE group_id = $1"
	row := r.db.QueryRowContext(ctx, query, localGroupID)
	var group domain.Group
	if err := row.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No group found with the given ID
		}
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) GetGroupsForPeer(ctx context.Context, localPeerID string) ([]*domain.Group, error) {
	// peerID - string is local id used
	var peerID int64
	query1 := `SELECT id FROM peers WHERE peer_id = $1`
	row := r.db.QueryRowContext(ctx, query1, localPeerID)
	if err := row.Scan(&peerID); err != nil {
		return nil, err
	}
	query := `SELECT id,name,user_id,group_id,created_at 
				FROM groups AS g
				 JOIN group_peer_membership AS m
				   ON g.id = m.group_id
				     WHERE m.peer_id = $1`
	rows, err := r.db.QueryContext(ctx, query, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*domain.Group
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var group domain.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.UserID, &group.GroupID, &group.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}
	return groups, nil
}

type DBId struct {
	PeerID  int64
	GroupID int64
}

func (r *GroupRepository) getDBPeerGroupID(ctx context.Context, localPeerID, localGroupID string) (DBId, error) {
	query1 := "SELECT id FROM peers WHERE peer_id = $1"
	var id int64
	row := r.db.QueryRowContext(ctx, query1, localPeerID)
	if err := row.Scan(&id); err != nil {
		return DBId{}, err
	}
	query2 := "SELECT id FROM groups WHERE group_id = $1"
	var groupID int64
	row = r.db.QueryRowContext(ctx, query2, localGroupID)
	if err := row.Scan(&groupID); err != nil {
		return DBId{}, err
	}
	return DBId{
		PeerID:  id,
		GroupID: groupID,
	}, nil
}

func (r *GroupRepository) CreateGroupPeerMembership(
	ctx context.Context,
	localPeerID, localGroupID string,
	role domain.PeerRole) error {
	dbIDs, err := r.getDBPeerGroupID(ctx, localPeerID, localGroupID)
	if err != nil {
		return err
	}
	query := "INSERT INTO group_peer_membership(group_id,peer_id,role) VALUES($1,$2,$3)"
	_, err = r.db.ExecContext(ctx, query, dbIDs.GroupID, dbIDs.PeerID, role.String())
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupRepository) GetGroupPeerMembership(
	ctx context.Context,
	localPeerID, localGroupID string) (domain.PeerGroupMembership, error) {
	dbIDs, err := r.getDBPeerGroupID(ctx, localPeerID, localGroupID)
	if err != nil {
		return domain.PeerGroupMembership{}, err
	}
	var mem domain.PeerGroupMembership
	query := "SELECT group_id,peer_id,role,joined_at FROM group_peer_membership WHERE group_id = $1 AND peer_id = $2"
	row := r.db.QueryRowContext(ctx, query, dbIDs.GroupID, dbIDs.PeerID)
	if err := row.Scan(&mem.GroupID, &mem.PeerID, &mem.Role, &mem.JoinedAt); err != nil {
		return domain.PeerGroupMembership{}, err
	}
	return mem, nil
}

func (r *GroupRepository) CreateJoinRequest(
	ctx context.Context,
	localGroupID string,
	localSenderPeerID string,
	localRecipientPeerID string,
	joinType domain.JoinType,
) error {
	query1 := "SELECT id FROM peers WHERE peer_id = $1"
	var senderPeerID int64
	row := r.db.QueryRowContext(ctx, query1, localSenderPeerID)
	if err := row.Scan(&senderPeerID); err != nil {
		return err
	}
	query2 := "SELECT id FROM peers WHERE peer_id = $1"
	var recipientPeerID int64
	row = r.db.QueryRowContext(ctx, query2, recipientPeerID)
	if err := row.Scan(&recipientPeerID); err != nil {
		return err
	}
	query3 := "SELECT id FROM groups WHERE group_id = $1"
	var groupID int64
	row = r.db.QueryRowContext(ctx, query3, localGroupID)
	if err := row.Scan(&groupID); err != nil {
		return err
	}
	query := `INSERT INTO membership_requests(group_id,sender_peer_id,recipient_peer_id,type) 
				VALUES($1,$2,$3,$4)`
	_, err := r.db.ExecContext(ctx, query, groupID, senderPeerID, recipientPeerID, joinType.String())
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupRepository) GetJoinRequest(
	ctx context.Context,
	requestID int64,
) (*domain.JoinRequest, error) {
	query := `
    SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE m.id = $1
`
	var j domain.JoinRequest
	row := r.db.QueryRowContext(ctx, query, requestID)
	if err := row.Scan(
		&j.RequestID,
		&j.GroupID,
		&j.GroupName,
		&j.SenderPeerID,
		&j.SenderPeerName,
		&j.RecipientPeerID,
		&j.RecipientPeerName,
		&j.JoinType, &j.JoinStatus, &j.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *GroupRepository) UpdateJoinRequestStatus(
	ctx context.Context,
	requestID int64,
	status domain.JoinStatus) error {
	query := `
			UPDATE membership_requests
				SET request_status = $1
			WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status.String(), requestID)
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupRepository) GetAllPendingInvites(
	ctx context.Context, peerID string) ([]domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE recipient.peer_id = $1
		AND 
		  m.request_status = $2
		    AND 
			  m.request_type = $3
			`
	rows, err := r.db.QueryContext(
		ctx,
		query, peerID,
		domain.PENDING.String(),
		domain.INVITE.String())
	if err != nil {
		return []domain.JoinRequest{}, err
	}
	defer rows.Close()
	pendingInviteRequests := make([]domain.JoinRequest, 0)
	for rows.Next() {
		var pendingInvite domain.JoinRequest
		if err := rows.Scan(
			&pendingInvite.RequestID,
			&pendingInvite.GroupID,
			&pendingInvite.GroupName,
			&pendingInvite.SenderPeerID,
			&pendingInvite.SenderPeerName,
			&pendingInvite.RecipientPeerID,
			&pendingInvite.RecipientPeerName,
			&pendingInvite.JoinType,
			&pendingInvite.JoinStatus,
			&pendingInvite.CreatedAt); err != nil {
			return []domain.JoinRequest{}, err
		}
		pendingInviteRequests = append(pendingInviteRequests, pendingInvite)
	}
	if err := rows.Err(); err != nil {
		return []domain.JoinRequest{}, err
	}
	return pendingInviteRequests, nil
}

func (r *GroupRepository) GetGroupPendingInvite(
	ctx context.Context, peerID string,
	groupID string) (*domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE recipient.peer_id = $1
			AND
			g.group_id = $2
			AND 
		 	m.request_status = $2
		    AND 
			m.request_type = $3	
			`
	var groupPendingRequest domain.JoinRequest
	row := r.db.QueryRowContext(
		ctx,
		query, peerID, groupID,
		domain.PENDING.String(),
		domain.INVITE.String())
	if err := row.Scan(
		&groupPendingRequest.RequestID,
		&groupPendingRequest.GroupID,
		&groupPendingRequest.GroupName,
		&groupPendingRequest.SenderPeerID,
		&groupPendingRequest.SenderPeerName,
		&groupPendingRequest.RecipientPeerID,
		&groupPendingRequest.RecipientPeerName,
		&groupPendingRequest.JoinType,
		&groupPendingRequest.JoinStatus,
		&groupPendingRequest.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &groupPendingRequest, nil
}

// in join req sender is the peer
func (r *GroupRepository) GetAllPendingJoinRequestsBySender(
	ctx context.Context, peerID string) ([]domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE sender.peer_id = $1
		AND 
		  m.request_status = $2
		    AND 
			  m.request_type = $3
			`
	rows, err := r.db.QueryContext(
		ctx,
		query, peerID,
		domain.PENDING.String(),
		domain.JOIN_REQUEST.String())
	if err != nil {
		return []domain.JoinRequest{}, err
	}
	defer rows.Close()
	pendingInviteRequests := make([]domain.JoinRequest, 0)
	for rows.Next() {
		var pendingInvite domain.JoinRequest
		if err := rows.Scan(
			&pendingInvite.RequestID,
			&pendingInvite.GroupID,
			&pendingInvite.GroupName,
			&pendingInvite.SenderPeerID,
			&pendingInvite.SenderPeerName,
			&pendingInvite.RecipientPeerID,
			&pendingInvite.RecipientPeerName,
			&pendingInvite.JoinType,
			&pendingInvite.JoinStatus,
			&pendingInvite.CreatedAt); err != nil {
			return []domain.JoinRequest{}, err
		}
		pendingInviteRequests = append(pendingInviteRequests, pendingInvite)
	}
	if err := rows.Err(); err != nil {
		return []domain.JoinRequest{}, err
	}
	return pendingInviteRequests, nil
}

// in join req sender is the peer and recipient is group owner
// these can be put in one func with status and peer id
// but for simplicity put this way
func (r *GroupRepository) GetAllPendingJoinRequestsForOwner(
	ctx context.Context, peerID string) ([]domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE recipient.peer_id = $1
		AND 
		  m.request_status = $2
		    AND 
			  m.request_type = $3
			`
	rows, err := r.db.QueryContext(
		ctx,
		query, peerID,
		domain.PENDING.String(),
		domain.JOIN_REQUEST.String())
	if err != nil {
		return []domain.JoinRequest{}, err
	}
	defer rows.Close()
	pendingInviteRequests := make([]domain.JoinRequest, 0)
	for rows.Next() {
		var pendingInvite domain.JoinRequest
		if err := rows.Scan(
			&pendingInvite.RequestID,
			&pendingInvite.GroupID,
			&pendingInvite.GroupName,
			&pendingInvite.SenderPeerID,
			&pendingInvite.SenderPeerName,
			&pendingInvite.RecipientPeerID,
			&pendingInvite.RecipientPeerName,
			&pendingInvite.JoinType,
			&pendingInvite.JoinStatus,
			&pendingInvite.CreatedAt); err != nil {
			return []domain.JoinRequest{}, err
		}
		pendingInviteRequests = append(pendingInviteRequests, pendingInvite)
	}
	if err := rows.Err(); err != nil {
		return []domain.JoinRequest{}, err
	}
	return pendingInviteRequests, nil
}

// get pending join req for a group owner from other peers of same group
// owner is recipient in case of join
func (r *GroupRepository) GetGroupPendingJoinRequestsForOwner(
	ctx context.Context,
	peerID string,
	groupID string,
) ([]domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE recipient.peer_id = $1
		AND g.group_id = $2
			AND 
		  m.request_status = $2
		    AND 
			  m.request_type = $3
			`
	rows, err := r.db.QueryContext(
		ctx,
		query, peerID, groupID,
		domain.PENDING.String(),
		domain.JOIN_REQUEST.String())
	if err != nil {
		return []domain.JoinRequest{}, err
	}
	defer rows.Close()
	pendingJoinRequests := make([]domain.JoinRequest, 0)
	for rows.Next() {
		var pendingJoin domain.JoinRequest
		if err := rows.Scan(
			&pendingJoin.RequestID,
			&pendingJoin.GroupID,
			&pendingJoin.GroupName,
			&pendingJoin.SenderPeerID,
			&pendingJoin.SenderPeerName,
			&pendingJoin.RecipientPeerID,
			&pendingJoin.RecipientPeerName,
			&pendingJoin.JoinType,
			&pendingJoin.JoinStatus,
			&pendingJoin.CreatedAt); err != nil {
			return []domain.JoinRequest{}, err
		}
		pendingJoinRequests = append(pendingJoinRequests, pendingJoin)
	}
	if err := rows.Err(); err != nil {
		return []domain.JoinRequest{}, err
	}
	return pendingJoinRequests, nil
}

// get join request of type join by peer for a particular group (at most 1
// join req can be pending for a group by a peer)
func (r *GroupRepository) GetGroupPendingJoinRequestsBySender(
	ctx context.Context,
	peerID string,
	groupID string) (*domain.JoinRequest, error) {
	query := `
	SELECT
        m.id AS request_id,
        g.group_id AS group_id,
        g.group_name AS group_name,
        sender.peer_id AS sender_peer_id,
        sender.name AS sender_peer_name,
        recipient.peer_id AS recipient_peer_id,
        recipient.name AS recipient_peer_name,
        m.request_type AS join_type,
        m.request_status AS join_status,
        m.created_at AS created_at
    FROM membership_requests m
    JOIN peers sender ON sender.id = m.sender_peer_id
    JOIN peers recipient ON recipient.id = m.recipient_peer_id
    JOIN groups g ON g.id = m.group_id
    WHERE sender.peer_id = $1
			AND
			g.group_id = $2
			AND 
		 	m.request_status = $2
		    AND 
			m.request_type = $3	
			`
	var groupPendingRequest domain.JoinRequest
	row := r.db.QueryRowContext(
		ctx,
		query, peerID, groupID,
		domain.PENDING.String(),
		domain.JOIN_REQUEST.String())
	if err := row.Scan(
		&groupPendingRequest.RequestID,
		&groupPendingRequest.GroupID,
		&groupPendingRequest.GroupName,
		&groupPendingRequest.SenderPeerID,
		&groupPendingRequest.SenderPeerName,
		&groupPendingRequest.RecipientPeerID,
		&groupPendingRequest.RecipientPeerName,
		&groupPendingRequest.JoinType,
		&groupPendingRequest.JoinStatus,
		&groupPendingRequest.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &groupPendingRequest, nil
}

// get group owner - peer id
func (r *GroupRepository) GetGroupRolePeerID(ctx context.Context, role domain.PeerRole, groupID string) (string, error) {
	query := `
		SELECT p.peer_id FROM 
			group_peer_membership m JOIN
				groups g
				  ON g.id = m.group_id
				JOIN peers p
				  ON p.id = m.peer_id
				   	WHERE g.group_id = $1
						AND m.role = $2
	`
	row := r.db.QueryRowContext(ctx, query, groupID, role.String())
	var peerID string
	if err := row.Scan(&peerID); err != nil {
		return "", err
	}
	return peerID, nil
}

// group id (int)  peer id (int)
// id (int)  peer id (string) name ....
// id (int)  group id(string). name ....

func (r *GroupRepository) GetPeerSyncSnapshot(ctx context.Context, peerID string) ([]domain.GroupWithPeers, error) {
	query := `
		SELECT
   			 g.id,
   			 g.group_id,
   			 g.name,
   			 g.created_at,
    	JSONB_AGG(
        JSONB_BUILD_OBJECT(
            'id', p.id,
            'peer_id', p.peer_id,
            'name', p.name,
            'address', p.address,
            'port', p.port,
            'last_seen', p.last_seen
        )
    	) AS peers
		FROM groups g
		JOIN group_peer_membership m ON m.group_id = g.id
		JOIN peers p ON p.id = m.peer_id
		WHERE g.id IN (
    		SELECT group_id
    		FROM group_peer_membership
   			WHERE peer_id = (
        	SELECT id FROM peers WHERE peer_id = $1
    	)
	)
	GROUP BY g.id, g.group_id, g.name, g.created_at;
	`

	rows, err := r.db.QueryContext(ctx, query, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.GroupWithPeers
	for rows.Next() {
		var group domain.GroupWithPeers

		if err := rows.Scan(
			&group.Id,
			&group.GroupId,
			&group.Name,
			&group.CreatedAt,
			&group.Peers,
		); err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}
