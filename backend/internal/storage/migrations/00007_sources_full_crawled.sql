-- +goose Up
ALTER TABLE sources ADD COLUMN IF NOT EXISTS full_crawled_at TIMESTAMPTZ;

-- Источники с уже загруженным каталогом считаем полностью пройденными:
-- при следующем старте краулер пойдёт в инкрементальный режим.
UPDATE sources s
SET full_crawled_at = now()
FROM manga m
WHERE m.parse_source_id = s.id
  AND s.full_crawled_at IS NULL;

-- +goose Down
ALTER TABLE sources DROP COLUMN IF EXISTS full_crawled_at;
