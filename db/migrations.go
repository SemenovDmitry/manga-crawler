package db

import (
	"embed"
	"log"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations запускает миграции базы данных
func RunMigrations() error {
	database := GetDB()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(database, "migrations"); err != nil {
		return err
	}

	log.Println("Миграции успешно применены")
	return nil
}
