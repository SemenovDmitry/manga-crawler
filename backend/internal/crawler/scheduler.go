package crawler

import (
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// fullCrawlDue — наступило ли время полного обхода по расписанию источника.
// Расписание живёт в БД (sources.next_full_crawl_at; NULL — обход нужен
// при первой возможности), поэтому у каждого источника оно своё и
// проверяется и при старте, и в каждом инкрементальном проходе.
func fullCrawlDue(src storage.Source, now time.Time) bool {
	if !src.NextFullCrawlAt.Valid {
		return true // источник ещё ни разу не обходился полностью
	}

	return !now.Before(src.NextFullCrawlAt.Time)
}

// needsFullCrawl решает, нужен ли источнику полный обход при старте
// (учитывает и ручной ForceFull).
func needsFullCrawl(cfg Config, src storage.Source, now time.Time) bool {
	if cfg.ForceFull {
		return true
	}

	return fullCrawlDue(src, now)
}
