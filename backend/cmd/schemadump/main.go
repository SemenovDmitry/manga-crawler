// cmd/schemadump — перегенерация снимка схемы БД в docs/schema.sql.
//
// Зачем: миграции разнесены по файлам, а читать схему целиком удобнее
// из одного файла (см. docs/schema.md). Снимок снимается pg_dump'ом из
// живой БД — контейнер db должен быть запущен, а миграции применены
// (они накатываются автоматически при старте бота/краулера).
//
// Кроссплатформенно: make и sed на Windows не требуются, достаточно
//
//	go run ./cmd/schemadump
//
// Утилита выполняет `docker compose exec db pg_dump ...` (клиент pg_dump
// есть в контейнере postgres), отбрасывает psql-специфику и пишет файл.
// Параметры БД берутся из .env/DB_* (они же прокидываются в контейнер).
//
// Выход помечается заголовком «сгенерировано, не редактировать руками»
// и перезаписывается только при фактическом изменении содержимого,
// чтобы не шуметь в git-status.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// outPath — снимок схемы относительно каталога с go.mod.
const outPath = "docs/schema.sql"

// header — заголовок снимка: назначение и способ перегенерации.
const header = `-- ============================================================================
-- Полная схема БД manga-crawler: таблицы, поля, индексы, связи (FK).
--
-- Сгенерировано командой: go run ./cmd/schemadump (не редактировать руками!)
-- Источник истины — goose-миграции в internal/storage/migrations/.
-- Человекочитаемый справочник: docs/schema.md.
-- ============================================================================`

func main() {
	log.SetFlags(0)

	out := flag.String("out", outPath, "путь к файлу снимка относительно каталога с go.mod")
	composeFile := flag.String("f", "docker-compose.yml", "compose-файл проекта")
	flag.Parse()

	// .env лежит рядом с docker-compose.yml — там же, откуда запускается утилита.
	_ = godotenv.Load()

	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	if user == "" || dbname == "" {
		log.Fatal("Не заданы DB_USER/DB_NAME (проверьте .env — docker compose exec читает их на хосте)")
	}

	dump, err := execPgDump(*composeFile, user, dbname)
	if err != nil {
		log.Fatalf("Не удалось снять схему: %v", err)
	}

	if err := writeSnapshot(*out, cleanupDump(dump)); err != nil {
		log.Fatalf("Не удалось записать снимок: %v", err)
	}
}

// execPgDump снимает DDL через pg_dump внутри контейнера db
// (эквивалент pg_dump --schema-only --no-owner --no-privileges).
func execPgDump(composeFile, user, dbname string) (string, error) {
	if _, err := os.Stat(composeFile); err != nil {
		return "", fmt.Errorf("%s не найден: запускайте из каталога backend", composeFile)
	}

	cmd := exec.Command(
		"docker", "compose", "-f", composeFile, "exec", "-T", "db",
		"pg_dump", "-U", user, "--schema-only", "--no-owner", "--no-privileges", dbname,
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(msg, "docker") && (strings.Contains(msg, "cannot") || strings.Contains(msg, "pipe") || strings.Contains(msg, "daemon")) {
			return "", fmt.Errorf("docker недоступен: %s — запустите Docker Desktop и контейнер db", msg)
		}
		return "", errors.New(msg)
	}

	if strings.TrimSpace(stdout.String()) == "" {
		return "", errors.New("pg_dump вернул пустой вывод")
	}
	return stdout.String(), nil
}

// reRestrict — psql-специфика нового pg_dump (\restrict <nonce>),
// ломающая файл вне psql-контекста.
var reRestrict = regexp.MustCompile(`(?m)^\\(un)?restrict .*$\n?`)

// cleanupDump выбрасывает psql-специфику и подрезает пробельный хвост.
func cleanupDump(dump string) string {
	return strings.TrimRight(reRestrict.ReplaceAllString(dump, ""), "\n") + "\n"
}

// writeSnapshot пишет снимок с заголовком; запись пропускается, если
// содержимое не изменилось.
func writeSnapshot(out, schema string) error {
	content := header + "\n\n" + schema

	if current, err := os.ReadFile(out); err == nil && string(current) == content {
		log.Printf("Схема не изменилась: %s актуален", out)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
		return err
	}

	log.Printf("Схема сохранена: %s", out)
	return nil
}
