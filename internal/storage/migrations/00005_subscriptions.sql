-- +goose Up
CREATE TABLE IF NOT EXISTS subscriptions (
    chat_id    BIGINT NOT NULL REFERENCES chat_users(chat_id) ON DELETE CASCADE,
    manga_id   INT    NOT NULL REFERENCES manga(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, manga_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_manga ON subscriptions (manga_id);

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
