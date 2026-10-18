-- +goose Up
CREATE TABLE IF NOT EXISTS chat_users (
    chat_id    BIGINT PRIMARY KEY,
    username   TEXT NOT NULL DEFAULT '',
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS chat_users;
