-- +goose Up
ALTER TABLE notifications ALTER COLUMN manga_id DROP NOT NULL;

-- +goose Down
DELETE FROM notifications WHERE manga_id IS NULL;
ALTER TABLE notifications ALTER COLUMN manga_id SET NOT NULL;
