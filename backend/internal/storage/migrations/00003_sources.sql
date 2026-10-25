-- +goose Up
CREATE TABLE IF NOT EXISTS sources (
    id       SERIAL PRIMARY KEY,
    name     TEXT NOT NULL UNIQUE,
    base_url TEXT NOT NULL
);

INSERT INTO sources (name, base_url) VALUES
    ('seimanga', 'https://1.seimanga.me'),
    ('zazaza',   'https://a.zazaza.me')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM sources WHERE name IN ('seimanga', 'zazaza');
DROP TABLE IF EXISTS sources;
