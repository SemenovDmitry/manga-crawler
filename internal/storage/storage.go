package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Store provides access to the database-backed repositories.
type Store struct {
	db *sql.DB
}

// NewStore opens the database connection described by the environment
// and returns a Store backed by it.
func NewStore() (*Store, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// DB exposes the underlying connection for infrastructure tasks
// such as migrations and ping checks.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initDB() (*sql.DB, error) {
	_ = godotenv.Load()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASSWORD")

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
