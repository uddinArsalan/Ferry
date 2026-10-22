DROP INDEX IF EXISTS one_pending_request_per_sender_group;

CREATE UNIQUE INDEX one_pending_invite_per_recipient_group
ON membership_requests (group_id, recipient_peer_id)
WHERE status = 'PENDING'
  AND type = 'INVITE';

CREATE UNIQUE INDEX one_pending_join_request_per_peer_group
ON membership_requests (group_id, sender_peer_id)
WHERE status = 'PENDING'
  AND type = 'JOIN_REQUEST';