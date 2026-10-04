CREATE TYPE request_type AS ENUM('JOIN_REQUEST','INVITE');
CREATE TYPE request_status AS ENUM('PENDING','ACCEPTED','REJECTED','CANCELLED');

CREATE TABLE membership_requests(
   id BIGSERIAL PRIMARY KEY,

    group_id BIGINT NOT NULL
        REFERENCES groups(id)
        ON DELETE CASCADE,

    type request_type NOT NULL,

    sender_peer_id BIGINT NOT NULL
        REFERENCES peers(id)
        ON DELETE CASCADE,

    recipient_peer_id BIGINT NOT NULL
        REFERENCES peers(id)
        ON DELETE CASCADE,

    status request_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX one_pending_request_per_sender_group
ON membership_requests (group_id, sender_peer_id)
WHERE status = 'PENDING';
-- one peer should have one pending request per group
