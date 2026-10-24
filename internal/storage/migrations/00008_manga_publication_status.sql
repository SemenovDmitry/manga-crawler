-- +goose Up
ALTER TABLE manga ADD COLUMN IF NOT EXISTS publication_status TEXT NOT NULL DEFAULT 'ongoing'
    CHECK (publication_status IN ('ongoing', 'completed', 'hiatus', 'cancelled', 'discontinued'));

CREATE INDEX IF NOT EXISTS idx_manga_publication_status ON manga (publication_status);

-- +goose Down
DROP INDEX IF EXISTS idx_manga_publication_status;
ALTER TABLE manga DROP COLUMN IF EXISTS publication_status;
