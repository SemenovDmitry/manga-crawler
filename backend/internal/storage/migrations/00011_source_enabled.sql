-- +goose Up
-- enabled — источник участвует в обходах; false = сервис временно
-- выключен (краулер пропускает источник без рестарта).
ALTER TABLE sources
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE sources
    DROP COLUMN enabled;
