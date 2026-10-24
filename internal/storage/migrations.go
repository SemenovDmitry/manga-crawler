package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// migrationLockID — произвольная константа advisory-блокировки Postgres.
// Бот и краулер применяют одни и те же миграции при старте; блокировка
// гарантирует, что одновременно их выполняет только один процесс.
const migrationLockID int64 = 0x6D434D47 // "MCMG"

// migrationLockTimeout — сколько ждать, если миграции уже выполняет другой
// процесс. По таймауту процесс завершится с ошибкой и restart-политика
// docker запустит его заново.
const migrationLockTimeout = 5 * time.Minute

// RunMigrations применяется и ботом, и краулером: миграции зашиты в бинарник
// (go:embed), поэтому порядок старта контейнеров не важен — кто первый,
// тот и применяет, второй видит "no migrations to run". Повторный вызов
// безвреден: goose ведёт журнал версий в таблице goose_db_version.
// Одновременный запуск сериализуется advisory-блокировкой (goose.Up сам
// блокировок не ставит).
func RunMigrations(database *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	conn, err := database.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("ошибка получения соединения для блокировки миграций: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), migrationLockTimeout)
	defer cancel()

	// Session-level advisory lock: живёт до явной разблокировки или до
	// разрыва соединения — краш процесса не оставит мёртвый лок.
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("ошибка захвата блокировки миграций: %w", err)
	}

	defer func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer unlockCancel()
		if _, err := conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationLockID); err != nil {
			log.Printf("Предупреждение: ошибка разблокировки миграций: %v", err)
		}
	}()

	if err := goose.Up(database, "migrations"); err != nil {
		return err
	}

	log.Println("Миграции успешно применены")
	return nil
}
