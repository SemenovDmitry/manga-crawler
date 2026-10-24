package crawler

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// badPageDumper сохраняет HTML страниц, которые парсер не смог разобрать
// (заглушки, новая разметка) — для диагностики. dir — базовый каталог
// дампов (Config.BadPagesDir); пустая строка — дампы отключены.
type badPageDumper struct {
	dir string
}

// dump сохраняет тело страницы в <dir>/<host>/bad-pages/<url-part>.html.
// Дамп перезаписывается при повторе, чтобы не копить мусор. Имя файла —
// до 80 символов url-части, спецсимволы заменены на «_».
func (d badPageDumper) dump(src storage.Source, urlPart string, body []byte, parseErr error) {
	if d.dir == "" {
		return
	}

	base, err := url.Parse(src.BaseURL)
	if err != nil {
		return
	}

	hostDir := filepath.Join(d.dir, base.Host, "bad-pages")
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		log.Printf("[%s] Не удалось создать каталог %s: %v", src.Name, hostDir, err)
		return
	}

	name := sanitizeFileName(urlPart, 80)

	header := fmt.Sprintf("<!-- url: %s\n     parse error: %v\n     saved: %s -->\n",
		src.BaseURL+"/"+urlPart, parseErr, time.Now().Format(time.RFC3339))

	file := filepath.Join(hostDir, name+".html")
	if err := os.WriteFile(file, append([]byte(header), body...), 0o644); err != nil {
		log.Printf("[%s] Не удалось сохранить дамп %s: %v", src.Name, urlPart, err)
		return
	}
	log.Printf("[%s] Дамп нераспознанной страницы сохранён: %s", src.Name, filepath.ToSlash(file))
}

// sanitizeFileName приводит строку к безопасному имени файла:
// буквы/цифры/дефис/подчёркивание остаются, остальное заменяется на «_»;
// длина ограничивается limit.
func sanitizeFileName(s string, limit int) string {
	if len(s) > limit {
		s = s[:limit]
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
