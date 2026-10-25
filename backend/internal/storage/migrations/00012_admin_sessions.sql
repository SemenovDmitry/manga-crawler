-- +goose Up
CREATE TABLE admin_sessions (
    token_hash TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_admin_sessions_expires ON admin_sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS admin_sessions;
