package parsers

// Parser — парсер страницы манги конкретного источника.
type Parser interface {
	ParseMangaCard(body []byte) (*MangaCard, error)
}

// SourceDriver — адаптер источника: парсинг карточки плюс правила URL.
// Реализуется в подпакетах (grouple, mgeko) и регистрируется в краулере
// по значению sources.engine.
type SourceDriver interface {
	Parser
	// IsMangaURL — является ли URL из sitemap страницей манги.
	IsMangaURL(rawURL string) bool
	// MangaURLPart — url-часть манги для БД ("asura_2024", "manga/slug");
	// она же ключ уникальности (parse_source_id, url).
	MangaURLPart(rawURL string) (string, error)
}

// CatalogLister — необязательная способность драйвера: перечисление карточек
// манги через пагинацию каталога. Нужно источникам, чей sitemap ограничен
// (a.zazaza.me кладёт в sitemap не больше 10000 URL при каталоге ~29k манг,
// подтвердено сравнением с «Всего» на /list). Драйверы без каталога
// (mgeko) эту способность не реализуют — краулер тогда работает только
// по sitemap.
type CatalogLister interface {
	// CatalogPageURL — адрес страницы каталога со смещением offset
	// (первая страница — offset 0).
	CatalogPageURL(baseURL string, offset int) string
	// CatalogPageSize — число карточек манги на одной странице каталога.
	CatalogPageSize() int
	// MangaURLsFromCatalog — URL карточек манги из HTML страницы каталога.
	MangaURLsFromCatalog(body []byte, baseURL string) []string
}
