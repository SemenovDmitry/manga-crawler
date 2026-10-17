-- Manga Crawler Backend - Database Schema
-- PostgreSQL 12+

-- ============================================
-- ТАБЛИЦЫ
-- ============================================

-- Пользователи Telegram
CREATE TABLE IF NOT EXISTS telegram_users (
    id BIGINT PRIMARY KEY,                              -- Telegram user ID (используется как chat_id)
    username TEXT,                                      -- @username
    first_name TEXT,                                    -- Имя
    last_name TEXT,                                     -- Фамилия
    is_active BOOLEAN DEFAULT TRUE,                     -- Активен ли пользователь
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,     -- Дата регистрации
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP      -- Дата последнего обновления
);

-- Источники (сайты для парсинга манги)
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,                              -- Уникальный идентификатор
    parser_name TEXT NOT NULL UNIQUE,                   -- Имя парсера (readmanga, mintmanga)
    base_url TEXT NOT NULL,                             -- Базовый URL сайта
    is_active BOOLEAN DEFAULT TRUE,                     -- Активен ли источник
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,     -- Дата добавления
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP      -- Дата последнего обновления
);

-- Манга
CREATE TABLE IF NOT EXISTS manga (
    id SERIAL PRIMARY KEY,                              -- Уникальный идентификатор
    source_id INT NOT NULL,                             -- ID источника
    url TEXT NOT NULL,                                  -- URL-часть манги (например, asura_2024)
    title TEXT NOT NULL,                                -- Название манги
    last_chapter_url TEXT,                              -- URL последней известной главы
    last_chapter_title TEXT,                            -- Название последней главы
    last_check_at TIMESTAMP,                            -- Время последней проверки
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,     -- Дата добавления
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,     -- Дата последнего обновления
    
    CONSTRAINT fk_manga_source FOREIGN KEY (source_id) 
        REFERENCES sources(id) ON DELETE CASCADE,
    CONSTRAINT uq_manga_source_url UNIQUE (source_id, url)
);

-- Подписки пользователей на мангу
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id SERIAL PRIMARY KEY,                              -- Уникальный идентификатор подписки
    user_id BIGINT NOT NULL,                            -- ID пользователя Telegram
    manga_id INT NOT NULL,                              -- ID манги
    notify BOOLEAN DEFAULT TRUE,                        -- Отправлять ли уведомления
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,     -- Дата подписки
    
    CONSTRAINT fk_subscription_user FOREIGN KEY (user_id) 
        REFERENCES telegram_users(id) ON DELETE CASCADE,
    CONSTRAINT fk_subscription_manga FOREIGN KEY (manga_id) 
        REFERENCES manga(id) ON DELETE CASCADE,
    CONSTRAINT uq_user_manga UNIQUE (user_id, manga_id)
);

-- Главы манги
CREATE TABLE IF NOT EXISTS chapters (
    id SERIAL PRIMARY KEY,                              -- Уникальный идентификатор главы
    manga_id INT NOT NULL,                              -- ID манги
    url TEXT NOT NULL,                                  -- URL главы
    title TEXT NOT NULL,                                -- Название главы
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  -- Дата обнаружения главы
    
    CONSTRAINT fk_chapter_manga FOREIGN KEY (manga_id) 
        REFERENCES manga(id) ON DELETE CASCADE,
    CONSTRAINT uq_chapter_manga_url UNIQUE (manga_id, url)
);

-- ============================================
-- ИНДЕКСЫ
-- ============================================

CREATE INDEX IF NOT EXISTS idx_manga_source_id ON manga(source_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_manga_id ON user_subscriptions(manga_id);
CREATE INDEX IF NOT EXISTS idx_chapters_manga_id ON chapters(manga_id);
CREATE INDEX IF NOT EXISTS idx_chapters_discovered_at ON chapters(discovered_at DESC);

-- ============================================
-- НАЧАЛЬНЫЕ ДАННЫЕ
-- ============================================

INSERT INTO sources (parser_name, base_url) VALUES 
    ('readmanga', 'https://a.zazaza.me'),
    ('mintmanga', 'https://1.seimanga.me')
ON CONFLICT (parser_name) DO NOTHING;
