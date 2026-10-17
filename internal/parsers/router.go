package parsers

import (
	"fmt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

// ParserFunc тип функции парсера
type ParserFunc func(telegramBot *telegram.TelegramBot, source types.Source, mangaList []types.Manga) error

// parsers маппинг SourceName -> парсер
var parsers = map[types.SourceName]ParserFunc{
	types.SourceReadmanga: ReadmangaParser,
	types.SourceMintmanga: ReadmangaParser, // mintmanga использует тот же парсер (RSS формат идентичен)
}

// GetParser возвращает функцию парсера по имени источника
func GetParser(sourceName types.SourceName) (ParserFunc, error) {
	parser, exists := parsers[sourceName]
	if !exists {
		return nil, fmt.Errorf("парсер для '%s' не найден", sourceName)
	}
	return parser, nil
}

// RunParser запускает парсер для указанного источника
func RunParser(telegramBot *telegram.TelegramBot, source types.Source, mangaList []types.Manga) error {
	parser, err := GetParser(source.ParserName)
	if err != nil {
		return err
	}
	return parser(telegramBot, source, mangaList)
}

// RegisterParser регистрирует новый парсер
func RegisterParser(sourceName types.SourceName, parser ParserFunc) {
	parsers[sourceName] = parser
}
