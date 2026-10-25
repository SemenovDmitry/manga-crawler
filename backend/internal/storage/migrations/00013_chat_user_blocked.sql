-- +goose Up
ALTER TABLE chat_users
    ADD COLUMN blocked BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE chat_users
    DROP COLUMN blocked;
