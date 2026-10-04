CREATE TYPE peer_role AS ENUM('OWNER','MEMBER','ADMIN');

CREATE TABLE group_peer_membership(
    group_id BIGINT NOT NULL REFERENCES groups(id),
    peer_id BIGINT NOT NULL REFERENCES peers(id),
    role PEER_ROLE NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, peer_id)
);