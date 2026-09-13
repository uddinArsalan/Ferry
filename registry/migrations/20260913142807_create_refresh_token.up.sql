BEGIN;

CREATE TABLE refresh_tokens(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    revoked_at TIMESTAMPTZ ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    replaced_by_token_id BIGINT REFERENCES refresh_tokens(id) ON DELETE SET NULL
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
create index idx_refresh_tokens_token_hash on refresh_tokens(token_hash);

COMMIT;