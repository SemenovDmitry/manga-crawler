package crawler

import (
	"fmt"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/flame"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/grouple"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/mangalib"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/mgeko"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/remanga"
)

func DriverFor(engine string) (parsers.SourceDriver, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "grouple":
		// Сайты на движке Grouple: seimanga, zazaza и их клоны
		return grouple.NewParser(), nil
	case "mgeko":
		return mgeko.NewParser(), nil
	case "flame":
		// Flame Comics (flamecomics.xyz): самописный Next.js, только sitemap
		return flame.NewParser(), nil
	case "remanga":
		// Remanga (remanga.org): Next.js App Router, главы догружаются из API
		return remanga.NewParser(), nil
	case "mangalib":
		// MangaLib (mangalib.me): Vue SPA с SSR для роботов, главы из API
		return mangalib.NewParser(), nil
	default:
		return nil, fmt.Errorf(
			"неизвестный движок источника %q: добавьте драйвер в internal/crawler/parsers и case в registry.go",
			engine,
		)
	}
}
