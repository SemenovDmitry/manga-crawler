package crawler

import (
	"fmt"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/grouple"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers/mgeko"
)

func DriverFor(engine string) (parsers.SourceDriver, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "grouple":
		// Сайты на движке Grouple: seimanga, zazaza и их клоны
		return grouple.NewParser(), nil
	case "mgeko":
		return mgeko.NewParser(), nil
	default:
		return nil, fmt.Errorf(
			"неизвестный движок источника %q: добавьте драйвер в internal/crawler/parsers и case в registry.go",
			engine,
		)
	}
}
