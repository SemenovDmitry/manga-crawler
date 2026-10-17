-- +goose Up

-- Пользователи Telegram
CREATE TABLE IF NOT EXISTS telegram_users (
    id BIGINT PRIMARY KEY,                          -- Telegram user ID (используется как chat_id)
    username TEXT,                                  -- @username
    first_name TEXT,                                -- Имя
    last_name TEXT,                                 -- Фамилия
    is_active BOOLEAN DEFAULT TRUE,                 -- Активен ли пользователь
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Источники (сайты для парсинга)
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,
    parser_name TEXT NOT NULL,                      -- readmanga, mintmanga
    base_url TEXT NOT NULL,                         -- https://a.zazaza.me
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parser_name)
);

-- Манга
CREATE TABLE IF NOT EXISTS manga (
    id SERIAL PRIMARY KEY,
    source_id INT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    url TEXT NOT NULL,                              -- URL-часть манги (например, asura_2024)
    title TEXT NOT NULL,                            -- Название манги
    last_chapter_url TEXT,                          -- URL последней известной главы
    last_chapter_title TEXT,                        -- Название последней главы
    last_check_at TIMESTAMP,                        -- Время последней проверки
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, url)
);

-- Подписки пользователей на мангу
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES telegram_users(id) ON DELETE CASCADE,
    manga_id INT NOT NULL REFERENCES manga(id) ON DELETE CASCADE,
    notify BOOLEAN DEFAULT TRUE,                    -- Отправлять уведомления
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, manga_id)
);

-- Главы манги
CREATE TABLE IF NOT EXISTS chapters (
    id SERIAL PRIMARY KEY,
    manga_id INT NOT NULL REFERENCES manga(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT NOT NULL,
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(manga_id, url)
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_manga_source_id ON manga(source_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_manga_id ON user_subscriptions(manga_id);
CREATE INDEX IF NOT EXISTS idx_chapters_manga_id ON chapters(manga_id);

-- Начальные данные источников
INSERT INTO sources (parser_name, base_url) VALUES 
    ('readmanga', 'https://a.zazaza.me'),
    ('mintmanga', 'https://1.seimanga.me')
ON CONFLICT (parser_name) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS user_subscriptions;
DROP TABLE IF EXISTS manga;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS telegram_users;
