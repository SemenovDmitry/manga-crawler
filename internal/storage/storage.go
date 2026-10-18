package storage

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

func GetConnection() *sql.DB {
	once.Do(func() {
		var err error
		conn, err = initDB()
		if err != nil {
			log.Fatalf("Ошибка подключения к БД: %v", err)
		}
	})
	return conn
}

func initDB() (*sql.DB, error) {
	_ = godotenv.Load()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

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

func Close() error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}
