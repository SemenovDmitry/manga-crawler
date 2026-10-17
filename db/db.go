package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	conn *sql.DB
	once sync.Once
)

// GetDB возвращает singleton подключение к базе данных
func GetDB() *sql.DB {
	once.Do(func() {
		var err error
		conn, err = initDB()
		if err != nil {
			log.Fatalf("Ошибка подключения к БД: %v", err)
		}
	})
	return conn
}

// initDB инициализирует подключение к PostgreSQL
func initDB() (*sql.DB, error) {
	// Загружаем .env если есть
	_ = godotenv.Load()

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "mangauser")
	password := getEnv("DB_PASSWORD", "mangapass")
	dbname := getEnv("DB_NAME", "mangadb")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения: %w", err)
	}

	// Проверяем соединение
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка пинга БД: %w", err)
	}

	log.Println("Подключение к базе данных успешно установлено")
	return database, nil
}

// CloseDB закрывает соединение с БД
func CloseDB() error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
