-- +goose Up
-- Подготовка к сервисным уведомлениям хозяину (очередь notifications
-- используется и для сообщений краулера без привязки к манге): manga_id
-- становится необязательным.
ALTER TABLE notifications ALTER COLUMN manga_id DROP NOT NULL;

-- +goose Down
DELETE FROM notifications WHERE manga_id IS NULL;
ALTER TABLE notifications ALTER COLUMN manga_id SET NOT NULL;
