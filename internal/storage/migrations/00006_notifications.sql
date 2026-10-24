-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id         BIGSERIAL PRIMARY KEY,
    chat_id    BIGINT NOT NULL,
    manga_id   INT    NOT NULL REFERENCES manga(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_pending
    ON notifications (created_at) WHERE sent_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS notifications;
