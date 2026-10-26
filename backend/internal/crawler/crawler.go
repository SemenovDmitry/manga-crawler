package crawler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

type crawlTarget struct {
	source storage.Source
	driver parsers.SourceDriver
}

type Crawler struct {
	store    repo
	cfg      Config
	fetch    *Fetcher
	pace     pacer
	notify   Notifier
	services ServiceNotifier // сервисные сообщения хозяину (автоотключение)
	badPages badPageDumper
	sitemap  urlLister // перечисление URL через sitemap источника
	catalog  urlLister // перечисление URL через пагинацию каталога
	gate     pauseGate // пауза фонового полного обхода на время инкрементального
	full     fullCrawlTracker

	// cardFailures — текущая серия ошибок парсинга карточек по источникам
	// (под failMu). Успешная карточка сбрасывает серию; порог
	// (Config.CardFailureLimit) отключает источник (processor.go).
	failMu       sync.Mutex
	cardFailures map[int]int
}

func New(store repo, cfg Config) *Crawler {
	fetch := NewFetcher()
	pace := pacer{delay: cfg.RequestDelay}

	var ownerChatID int64
	if cfg.OwnerChatID != "" {
		if id, err := strconv.ParseInt(cfg.OwnerChatID, 10, 64); err == nil {
			ownerChatID = id
		} else {
			log.Printf("Некорректный OwnerChatID %q: %v — сервисные сообщения отправляться не будут", cfg.OwnerChatID, err)
		}
	}

	c := &Crawler{
		store:        store,
		cfg:          cfg,
		fetch:        fetch,
		pace:         pace,
		notify:       NewQueueNotifier(store, ownerChatID),
		badPages:     badPageDumper{dir: cfg.BadPagesDir},
		cardFailures: make(map[int]int),
	}
	c.services, _ = c.notify.(ServiceNotifier)
	c.sitemap = newSitemapLister(fetch, &c.gate)
	c.catalog = newCatalogLister(fetch, pace, &c.gate)
	return c
}

func (c *Crawler) Run(ctx context.Context) {
	sources, err := c.crawlTargets()
	if err != nil {
		log.Fatalf("Краулер остановлен: %v", err)
	}
	if len(sources) == 0 {
		log.Fatal("Включённых источников нет в БД (sources.enabled = false)")
	}

	log.Printf(
		"Краулер запущен: источников %d, интервал %s, пауза между запросами %s",
		len(sources), c.cfg.Interval, c.cfg.RequestDelay,
	)

	c.logDatabaseStats()

	if c.cfg.Once {
		c.runStartupPasses(ctx, sources)
		log.Println("Разовый проход завершён, выход")
		return
	}

	c.runStartupPasses(ctx, sources)

	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Краулер остановлен")
			return
		case <-ticker.C:
			c.runIncrementalPars(ctx)
		}
	}
}

func (c *Crawler) runStartupPasses(ctx context.Context, srcs []crawlTarget) {
	backgroundFull := !c.cfg.Once
	var dueFull []crawlTarget

	for i := range srcs {
		if ctx.Err() != nil {
			return
		}
		src := &srcs[i]

		full := needsFullCrawl(c.cfg, src.source, time.Now())
		if full && c.cfg.SkipInitialFull {
			log.Printf("[%s] Полный обход пропущен (SkipInitialFull)", src.source.Name)
			full = false
		}
		if !full && src.source.NextFullCrawlAt.Valid {
			log.Printf(
				"[%s] Полный обход не требуется: следующий %s",
				src.source.Name,
				src.source.NextFullCrawlAt.Time.Format("02.01.2006 15:04"),
			)
		}

		switch {
		case full && backgroundFull:
			dueFull = append(dueFull, *src)
			c.crawlSource(ctx, src, false)
		default:
			c.crawlSource(ctx, src, full)
		}
	}

	// Первый из источников, требующих полного обхода, стартует в фоне сразу;
	// остальные подхватятся в инкрементальных проходах по расписанию.
	if len(dueFull) > 0 {
		c.startFullCrawl(ctx, dueFull[0])
	}

	c.logDatabaseStats()
}

// runIncrementalPars — инкрементальный проход по всем источникам.
// Фоновый полный обход на время прохода ставится на паузу (он замирает
// между двумя запросами к сайту), поэтому свежие главы не ждут его
// окончания. Полные обходы по расписанию запускаются в фоне, по одному.
func (c *Crawler) runIncrementalPars(ctx context.Context) {
	log.Println("=== Начало инкрементального прохода ===")

	sources, err := c.crawlTargets()
	if err != nil {
		log.Printf("Проход пропущен: %v", err)
		return
	}
	if len(sources) == 0 {
		log.Println("Все источники выключены (sources.enabled = false), краулер простаивает")
		return
	}

	busy, _, done := c.full.state()
	if busy {
		ack := c.gate.pause()
		select {
		case <-ack:
			log.Println("Фоновый полный обход приостановлен на время инкрементального прохода")
		case <-done:
			log.Println("Фоновый полный обход завершился сам, пауза не потребовалась")
		case <-ctx.Done():
			c.gate.resume()
			return
		}
	}
	// resume идемпотентен и будит приостановленный полный обход после прохода.
	defer c.gate.resume()

	var dueFull []crawlTarget
	for i := range sources {
		if ctx.Err() != nil {
			return
		}
		src := &sources[i]

		if fullCrawlDue(src.source, time.Now()) {
			if busy {
				log.Printf("[%s] Полный обход отложен: уже выполняется другой", src.source.Name)
			} else {
				dueFull = append(dueFull, *src)
			}
		}

		// Инкрементальная проверка выполняется для всех источников, включая
		// тот, чей полный обход стоит на паузе: запросы всё равно идут
		// строго по одному — из этого потока.
		c.crawlSource(ctx, src, false)
	}

	c.logDatabaseStats()
	log.Println("=== Инкрементальный проход завершён ===")

	if len(dueFull) > 0 {
		c.startFullCrawl(ctx, dueFull[0])
	}
}

// startFullCrawl запускает фоновый полный обход источника; одновременно
// выполняется не более одного. Между тиками он работает без пауз, а на
// время инкрементальных проходов встаёт на паузу (см. pauseGate).
func (c *Crawler) startFullCrawl(ctx context.Context, src crawlTarget) {
	if !c.full.start(src.source.Name) {
		log.Printf("[%s] Полный обход отложен: уже выполняется другой", src.source.Name)
		return
	}
	go func() {
		defer c.full.finish()
		c.crawlSource(ctx, &src, true)
	}()
}

func (c *Crawler) crawlSource(ctx context.Context, src *crawlTarget, full bool) {
	passType := "инкрементального"
	if full {
		passType = "полного"
	}
	log.Printf("[%s] Начало %s прохода", src.source.Name, passType)

	var err error
	if full {
		err = c.crawlSourceFull(ctx, *src)
		// Автоотключение прерывает полный обход: источник больше не включён,
		// расписание не отмечаем — при повторном включении обход начнётся заново.
		if err == nil && src.source.Enabled {
			next, markErr := c.store.MarkSourceFullCrawled(src.source.ID)
			if markErr != nil {
				log.Printf("[%s] Ошибка отметки полного обхода: %v", src.source.Name, markErr)
			} else {
				// Обновляем расписание в памяти: следующий полный обход —
				// через персональный интервал источника
				src.source.FullCrawledAt = sql.NullTime{Time: time.Now(), Valid: true}
				src.source.NextFullCrawlAt = next
				if next.Valid {
					log.Printf("[%s] Следующий полный обход: %s",
						src.source.Name, next.Time.Format("02.01.2006 15:04"))
				}
			}
		}
	} else {
		err = c.crawlSourceUpdates(ctx, *src)
	}

	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		var disabledErr errSourceDisabled
		if errors.As(err, &disabledErr) {
			// Источник отключён автоматически: уже уведомлены, не сбой.
			log.Printf("[%s] %s", src.source.Name, err)
		} else {
			log.Printf("[%s] Ошибка обхода: %v", src.source.Name, err)
		}
	}

	log.Printf("[%s] %s проход завершён", src.source.Name, passType)
}

// pacedPageLoader возвращает функцию загрузки страниц для догрузки данных
// драйвером (parsers.ChapterLoader): Fetcher.Page плюс пауза вежливости
// перед каждым запросом, чтобы API источника не нагружался.
func (c *Crawler) pacedPageLoader() parsers.PageLoader {
	return func(ctx context.Context, rawURL string) ([]byte, error) {
		if err := c.pace.wait(ctx); err != nil {
			return nil, err
		}
		return c.fetch.Page(ctx, rawURL)
	}
}

func (c *Crawler) crawlSourceFull(ctx context.Context, src crawlTarget) error {
	urls, err := c.sitemap.list(ctx, src, 0)
	if err != nil {
		return err
	}
	log.Printf("[%s] В sitemap найдено страниц манги: %d", src.source.Name, len(urls))

	// Каталог может обраться не полностью (сайт нестабилен) — используем
	// собранную часть, недостающее доберут инкрементальные проходы.
	catalogURLs, catalogErr := c.catalog.list(ctx, src, 0)
	switch {
	case len(catalogURLs) == 0:
		log.Printf("[%s] Каталог карточек не дал (%v), обхожу только sitemap", src.source.Name, catalogErr)
	case catalogErr != nil:
		log.Printf("[%s] Каталог обошёлся не полностью (%v), использую собранные %d карточек",
			src.source.Name, catalogErr, len(catalogURLs))
	default:
		log.Printf("[%s] Из каталога получено карточек: %d", src.source.Name, len(catalogURLs))
	}

	// В режиме продолжения — множество уже сохранённых url-частей;
	// в режиме полного обновления множество пустое и ничего не пропускается.
	known := urlSet{}
	if !c.cfg.ForceFull && !src.source.NextFullCrawlAt.Valid {
		if known, err = c.knownURLParts(src.source.ID); err != nil {
			return err
		}
	}

	// Sitemap: спарсенные (и уже известные в БД) карточки отмечаются в known —
	// через каталог они повторно не парсятся.
	res, err := c.processNewURLs(ctx, src, "Прогресс", urls, known)
	if err != nil {
		return err
	}
	if res.skipped > 0 {
		log.Printf("[%s] Пропущено уже известных манг: %d", src.source.Name, res.skipped)
	}

	// Каталог: парсим только карточки, которых не было в sitemap этого прохода.
	catalogRes, err := c.processNewURLs(ctx, src, "Каталог", catalogURLs, known)
	if err != nil {
		return err
	}
	if catalogRes.skipped > 0 {
		log.Printf("[%s] В каталоге пропущено уже спарсенных через sitemap: %d", src.source.Name, catalogRes.skipped)
	}
	if catalogRes.processed > 0 {
		log.Printf("[%s] Из каталога добавлено манг, которых не было в sitemap: %d", src.source.Name, catalogRes.processed)
	}

	return nil
}

func (c *Crawler) crawlSourceUpdates(ctx context.Context, src crawlTarget) error {
	stale, err := c.store.GetStaleSubscribedManga(
		src.source.ID,
		int(c.cfg.Interval.Seconds()),
		c.cfg.MangaRefreshLimit,
	)
	if err != nil {
		return fmt.Errorf("ошибка получения устаревших подписок: %w", err)
	}

	for _, manga := range stale {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rawURL := strings.TrimRight(src.source.BaseURL, "/") + "/" + strings.TrimLeft(manga.URL, "/")
		if err := c.processManga(ctx, src, rawURL); err != nil {
			log.Printf("[%s] Ошибка обновления %s: %v", src.source.Name, manga.URL, err)
		}

		if err := c.pace.wait(ctx); err != nil {
			return err
		}
	}
	if len(stale) > 0 {
		log.Printf("[%s] Проверено подписанных манг: %d", src.source.Name, len(stale))
	}

	return nil
}

func (c *Crawler) logDatabaseStats() {
	counts, err := c.store.GetMangaCountBySource()
	if err != nil {
		log.Printf("Ошибка подсчёта статистики: %v", err)
		return
	}
	for name, count := range counts {
		log.Printf("В БД манг {%s}: %d", name, count)
	}
}

func (c *Crawler) crawlTargets() ([]crawlTarget, error) {
	sources, err := c.store.GetSources(storage.EnabledOnly())
	if err != nil {
		return nil, fmt.Errorf("ошибка получения источников: %w", err)
	}

	targets := make([]crawlTarget, 0, len(sources))
	for _, source := range sources {
		driver, err := DriverFor(source.Engine)
		if err != nil {
			log.Printf("Источник %s (%s): %v — пропускаю", source.Name, source.BaseURL, err)
			continue
		}
		targets = append(targets, crawlTarget{source: source, driver: driver})
	}

	return targets, nil
}
