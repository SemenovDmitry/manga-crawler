-- +goose Up
CREATE TABLE IF NOT EXISTS manga (
    id              SERIAL PRIMARY KEY,
    parse_source_id INT NOT NULL REFERENCES sources(id),
    url             TEXT NOT NULL,
    title           TEXT NOT NULL,
    alt_titles      TEXT NOT NULL DEFAULT '',
    latest_chapter  TEXT NOT NULL DEFAULT '',
    chapters_count  INT NOT NULL DEFAULT 0,
    last_check_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (parse_source_id, url)
);

CREATE INDEX IF NOT EXISTS idx_manga_title_lower ON manga (lower(title));
CREATE INDEX IF NOT EXISTS idx_manga_alt_titles_lower ON manga (lower(alt_titles));

-- +goose Down
DROP TABLE IF EXISTS manga;
