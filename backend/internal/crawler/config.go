package crawler

import (
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/env"
)

type Config struct {
	Interval          time.Duration // пауза между проходами (по умолчанию 30 минут)
	RequestDelay      time.Duration // пауза между запросами к сайту (вежливость)
	MangaRefreshLimit int           // сколько подписанных манг обновлять за инкрементальный проход
	SkipInitialFull   bool          // не выполнять полный обход даже когда подошло расписание
	ForceFull         bool          // принудительный полный обход всех источников при старте
	Once              bool          // выполнить один проход и выйти (режим для cron)
	// BadPagesDir — каталог для дампов страниц, которые парсер не смог
	// разобрать (<dir>/<host>/bad-pages/). Пустая строка — дампы отключены.
	BadPagesDir string
}

func ConfigFromEnv() Config {
	return Config{
		Interval:          env.Duration("CRAWL_INTERVAL", 30*time.Minute),
		RequestDelay:      env.Duration("CRAWL_REQUEST_DELAY", time.Second),
		MangaRefreshLimit: env.Int("CRAWL_MANGA_REFRESH_LIMIT", 200),
		SkipInitialFull:   env.Bool("CRAWL_SKIP_FULL", false),
		ForceFull:         env.Bool("CRAWL_FORCE_FULL", false),
		Once:              env.Bool("CRAWL_ONCE", false),
		BadPagesDir:       env.String("CRAWL_BAD_PAGES_DIR", "probe-output"),
	}
}
