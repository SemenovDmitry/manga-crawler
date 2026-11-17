package storage

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/env"
	_ "github.com/lib/pq"
)

type Store struct {
	db *sql.DB
}

// NewStore подключается к Postgres (параметры — переменные окружения DB_*,
// .env загружает запускающий процесс) и применяет миграции.
func NewStore() (*Store, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка миграций: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initDB() (*sql.DB, error) {
	host := env.String("DB_HOST", "")
	port := env.String("DB_PORT", "")
	user := env.String("DB_USER", "")
	dbname := env.String("DB_NAME", "")
	password := env.String("DB_PASSWORD", "")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка пинга БД: %w", err)
	}

	log.Println("Подключение к базе данных успешно установлено")
	return database, nil
}
