-- +goose Up
ALTER TABLE sources
    ADD COLUMN IF NOT EXISTS full_crawl_interval_secs BIGINT NOT NULL DEFAULT 1209600, -- 2 недели
    ADD COLUMN IF NOT EXISTS next_full_crawl_at TIMESTAMPTZ;

-- Расписание для уже полностью обойдённых источников:
-- следующий полный обход через их интервал после последнего.
UPDATE sources
SET next_full_crawl_at = full_crawled_at + make_interval(secs => full_crawl_interval_secs)
WHERE full_crawled_at IS NOT NULL
  AND next_full_crawl_at IS NULL;

-- +goose Down
ALTER TABLE sources
    DROP COLUMN IF EXISTS next_full_crawl_at,
    DROP COLUMN IF EXISTS full_crawl_interval_secs;
