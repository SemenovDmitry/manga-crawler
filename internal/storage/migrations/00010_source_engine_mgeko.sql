-- +goose Up
-- Движок сайта определяет драйвер краулера (internal/crawler/registry.go):
-- правила отбора URL из sitemap и парсинга карточки.
ALTER TABLE sources
    ADD COLUMN IF NOT EXISTS engine TEXT NOT NULL DEFAULT 'grouple';

-- mgeko.cc — сайт на собственном движке (driver_mgeko.go)
INSERT INTO sources (name, base_url, engine)
VALUES ('mgeko', 'https://www.mgeko.cc', 'mgeko')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM sources WHERE name = 'mgeko';
ALTER TABLE sources DROP COLUMN IF EXISTS engine;
