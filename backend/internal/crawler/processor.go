package crawler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/crawler/parsers"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// urlSet — множество url-частей (ключей карточек): известные из БД плюс
// уже учтённые в текущем проходе. Гарантирует, что карточка не парсится
// дважды, даже если пришла и из sitemap, и из каталога.
type urlSet map[string]bool

func (s urlSet) contains(urlPart string) bool { return s[urlPart] }
func (s urlSet) add(urlPart string)           { s[urlPart] = true }

// processResult — итог обработки списка URL.
type processResult struct {
	processed int // карточек, успешно загруженных и сохранённых
	skipped   int // пропущено: уже известны или url-часть не распознана
	failed    int // карточек, не обработанных из-за ошибки (сеть, HTTP-статус, парсинг)
}

// cardParseError — не удалось разобрать разметку карточки: драйвер сломан,
// сайт отдал заглушку или изменил вёрстку. Отличается от сетевого сбоя:
// серия ошибок обработки, закончившаяся ошибкой парсинга, отключает
// источник, а не только останавливает проход (см. Crawler.noteCardFailure).
type cardParseError struct{ err error }

func (e *cardParseError) Error() string {
	return "ошибка парсинга карточки: " + e.err.Error()
}
func (e *cardParseError) Unwrap() error { return e.err }

func isCardParseError(err error) bool {
	var parseErr *cardParseError
	return errors.As(err, &parseErr)
}

// errCardSkipped — карточка пропущена без обработки: тайтла по адресу нет
// (HTTP 404 — удалён или ещё не в API). Не сбой (серию ошибок не растит,
// проход не останавливает) и не успех (серию пропусков не сбрасывает).
type errCardSkipped struct{}

func (e *errCardSkipped) Error() string {
	return "карточка пропущена: недоступна (404)"
}

func isCardSkipped(err error) bool {
	var skipped *errCardSkipped
	return errors.As(err, &skipped)
}

// knownURLParts — множество url-частей манг источника, сохранённых в БД.
func (c *Crawler) knownURLParts(sourceID int) (urlSet, error) {
	known, err := c.store.GetMangaURLs(sourceID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения известных манг: %w", err)
	}

	set := make(urlSet, len(known))
	for _, urlPart := range known {
		set.add(urlPart)
	}

	return set, nil
}

// processNewURLs обрабатывает карточки, url-части которых ещё не встречались
// в множестве seen (обновляется по ходу: см. urlSet). Ошибки отдельных
// карточек логируются; серия ошибок длиной Config.CardFailureLimit
// останавливает проход — на ошибке парсинга источник отключается
// (noteCardFailure), иначе просто прерывается до следующего прохода.
// Между обработками выдерживается пауза вежливости (pacer); label — имя
// канала для логов прогресса.
func (c *Crawler) processNewURLs(
	ctx context.Context, src crawlTarget, label string, urls []string, seen urlSet,
) (processResult, error) {
	var res processResult

	for i, rawURL := range urls {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}

		urlPart, partErr := src.driver.MangaURLPart(rawURL)
		if partErr != nil || seen.contains(urlPart) {
			res.skipped++
			continue
		}
		seen.add(urlPart)

		err := c.processManga(ctx, src, rawURL)
		switch {
		case err == nil:
			res.processed++
			c.noteCardSuccess(src.source.ID)
		case isCardSkipped(err):
			// 404-пропуск: ни сбой, ни успех — серии не трогает
			// (учёт серии пропусков внутри processManga).
		default:
			res.failed++
			log.Printf("[%s] Ошибка обработки %s: %v", src.source.Name, rawURL, err)
			stop, disabled := c.noteCardFailure(src, err)
			if disabled {
				return res, errSourceDisabled{source: src.source.Name}
			}
			if stop {
				return res, errPassAborted{source: src.source.Name}
			}
		}

		if err := c.pace.wait(ctx); err != nil {
			return res, err
		}
		// Точка уступки: на время инкрементального прохода полный обход
		// замирает между карточками (см. pauseGate).
		if err := c.gate.yield(ctx); err != nil {
			return res, err
		}
		if (i+1)%100 == 0 {
			log.Printf("[%s] %s: %d/%d", src.source.Name, label, i+1, len(urls))
		}
	}

	return res, nil
}

// cardErrorWarnShare — доля неудачных карточек в проходе (%), с которой
// сбои перестают считаться разовыми и логируются предупреждением.
const cardErrorWarnShare = 10

// logCardFailures сводит сбои обработки карточек за проход в одну строку.
// Массовые сбои (доля от cardErrorWarnShare и выше) — предупреждение:
// полный обход с легом сайта формально завершится успешно и отметит
// расписание, а пропущенные карточки будут ждать следующего прохода.
func logCardFailures(source, passKind string, attempted, failed int) {
	if failed == 0 || attempted <= 0 {
		return
	}

	share := 100 * failed / attempted
	if share >= cardErrorWarnShare {
		log.Printf(
			"[%s] ВНИМАНИЕ: сбои обработки массовые — не обработано %d карточек из %d (%d%%). Неудачные карточки повторятся при следующем %s; устойчивый уровень говорит о недоступности сайта",
			source, failed, attempted, share, passKind,
		)
		return
	}
	log.Printf("[%s] Не обработано карточек из-за ошибок: %d из %d (повторятся при следующем %s)",
		source, failed, attempted, passKind)
}

// errSourceDisabled — обход источника прерван его автоотключением
// (серия ошибок, закончившаяся ошибкой парсинга). Не логируется как сбой.
type errSourceDisabled struct{ source string }

func (e errSourceDisabled) Error() string {
	return "источник " + e.source + " отключён после серии ошибок парсинга"
}

// errPassAborted — проход остановлен серией ошибок обработки карточек
// (сайт или API недоступны). Источник остаётся включённым: следующий
// проход по расписанию повторит пропущенное, при восстановлении сайта
// всё восстановится само. Не логируется как сбой.
type errPassAborted struct{ source string }

func (e errPassAborted) Error() string {
	return "источник " + e.source + ": проход остановлен после серии ошибок обработки карточек"
}

// noteCardSuccess сбрасывает серию ошибок обработки источника:
// успешная карточка — сайт жив и разметка на месте. Заодно разрешает
// новое сообщение об остановке прохода (эпизод недоступности кончился).
func (c *Crawler) noteCardSuccess(sourceID int) {
	c.failMu.Lock()
	defer c.failMu.Unlock()
	delete(c.cardFailures, sourceID)
	delete(c.stopNotified, sourceID)
}

// noteCardFailure учитывает ошибку обработки карточки в серии источника
// (считаются любые ошибки: HTTP-статус, сеть, парсинг; сбрасывается серия
// успешной карточкой). Когда серия достигает Config.CardFailureLimit
// (0 — контроль выключен), возвращается stop — проход останавливается.
// Если серия закончилась ошибкой парсинга, источник дополнительно
// отключается в БД с сервисным сообщением хозяину (disable): сломанную
// разметку чинит человек, сетевые сбои проходят сами.
func (c *Crawler) noteCardFailure(src crawlTarget, err error) (stop, disable bool) {
	limit := c.cfg.CardFailureLimit
	if limit <= 0 {
		return false, false
	}

	c.failMu.Lock()
	c.cardFailures[src.source.ID]++
	consecutive := c.cardFailures[src.source.ID]
	c.failMu.Unlock()

	if consecutive < limit {
		return false, false
	}

	if !isCardParseError(err) {
		log.Printf(
			"[%s] %d ошибок обработки карточек подряд — останавливаю проход до следующего расписания (сайт или API недоступны?)",
			src.source.Name, consecutive,
		)

		// Сообщение хозяину — одно на эпизод недоступности: пока серия
		// не сброшена успехом, каждый следующий проход остановится так же,
		// и дублирующие сообщения завалили бы чат.
		c.failMu.Lock()
		firstStop := !c.stopNotified[src.source.ID]
		c.stopNotified[src.source.ID] = true
		c.failMu.Unlock()
		if firstStop && c.services != nil {
			c.services.ServiceMessage(formatSourceStoppedMessage(
				SourceInfo{Name: src.source.Name, BaseURL: src.source.BaseURL},
				consecutive,
				err,
			))
		}
		return true, false
	}

	log.Printf(
		"[%s] Порог ошибок парсинга достигнут (%d подряд) — отключаю источник",
		src.source.Name, consecutive,
	)

	if _, err := c.store.SetSourceEnabled(src.source.ID, false); err != nil {
		// Отключить не удалось: проходы продолжатся, ошибки будут копиться,
		// попытка отключения повторится на следующем достижении порога.
		// Сервисное сообщение не дублируем — ждём успешного отключения.
		log.Printf("[%s] ОШИБКА АВТООТКЛЮЧЕНИЯ: %v", src.source.Name, err)
		return false, false
	}
	src.source.Enabled = false

	if c.services != nil {
		c.services.ServiceMessage(formatSourceDisabledMessage(
			SourceInfo{Name: src.source.Name, BaseURL: src.source.BaseURL},
			consecutive,
			fmt.Errorf("см. свежие дампы bad-pages источника"),
		))
	}

	return true, true
}

// processManga загружает страницу манги, обновляет данные в БД
// и ставит уведомления подписчикам, если появились новые главы.
func (c *Crawler) processManga(ctx context.Context, src crawlTarget, rawURL string) error {
	urlPart, err := src.driver.MangaURLPart(rawURL)
	if err != nil {
		return err
	}

	state, found, err := c.store.GetMangaState(src.source.ID, urlPart)
	if err != nil {
		return err
	}

	// Драйвер может брать данные карточки из API источника (сайт за
	// антибот-защитой — parsers.CardRequestProvider), иначе грузится
	// страница сайта, возможно под роботным UA (parsers.CardUserAgent).
	var body []byte
	if rp, ok := src.driver.(parsers.CardRequestProvider); ok {
		apiURL, headers := rp.CardRequest(urlPart)
		body, err = c.fetch.Page(ctx, apiURL, headers...)
	} else {
		var uaOverride []string
		if ua, ok := src.driver.(parsers.CardUserAgent); ok {
			uaOverride = []string{ua.CardUserAgent()}
		}
		body, err = c.fetch.Card(ctx, rawURL, uaOverride...)
	}
	if err != nil {
		// Тайтл переехал на зеркало: редирект не переходим, карточка
		// пропускается — она уже учтена по новому адресу, если тот
		// встретится в sitemap или каталоге.
		if errors.Is(err, ErrRedirected) {
			log.Printf("[%s] Пропущен кросс-доменный редирект: %v", src.source.Name, err)
			return nil
		}
		// 404 — тайтла по этому адресу нет (удалён; у mangalib — ещё и
		// свежие тайтлы, которых нет в API). Пропуск, а не сбой: суть
		// ответа стабильна, серией таких карточек проход останавливать
		// нельзя — обход никогда не дошёл бы до живых тайтлов. Уведомлений
		// хозяину нет: пропуски — норма, их количество видно в логе.
		var statusErr *httpStatusError
		if errors.As(err, &statusErr) && statusErr.code == http.StatusNotFound {
			log.Printf("[%s] Карточка недоступна (404), пропускаю: %s", src.source.Name, rawURL)
			return &errCardSkipped{}
		}
		return err
	}

	card, err := src.driver.ParseMangaCard(body)
	if err != nil {
		if errors.Is(err, parsers.ErrMangaRemoved) {
			log.Printf("[%s] Манга удалена или скрыта, пропускаю: %s", src.source.Name, rawURL)
			return nil
		}
		// Неизвестная разметка/заглушка: сохраняем тело для разбора,
		// чтобы диагностировать причину, и сигнализируем специальным
		// типом ошибки — она учитывается автоотключением источника.
		c.badPages.dump(src.source, urlPart, body, err)
		return &cardParseError{err: err}
	}

	// Догрузка глав для драйверов, где список глав не в HTML карточки
	// (remanga): ошибка догрузки не отменяет карточку — название и статус
	// уже распарсены, главы доберутся при следующем обновлении.
	var loadErr error
	if loader, ok := src.driver.(parsers.ChapterLoader); ok {
		load := c.pacedPageLoader()
		loadErr = loader.LoadChapters(ctx, load, urlPart, card)
		if loadErr != nil {
			log.Printf("[%s] Догрузка глав %s не удалась: %v", src.source.Name, urlPart, loadErr)
		}
	}

	chapters := parsers.NormalizeChapters(card.Chapters)
	latestChapter := ""
	if len(chapters) > 0 {
		latestChapter = chapters[len(chapters)-1].Number
	}

	// Сбой догрузки — главы за этот проход неполные: в БД оставляем прежние
	// chapters_count/latest_chapter (storage.KeepChaptersCount), иначе сбой
	// выглядел бы как исчезновение глав, а после восстановления API весь
	// список ушёл бы подписчикам как «новые».
	chaptersCount := len(chapters)
	if loadErr != nil {
		chaptersCount = storage.KeepChaptersCount
	}

	mangaID, err := c.store.UpsertManga(
		src.source.ID, urlPart, card.Title, card.AltTitles, latestChapter,
		storage.PublicationStatus(card.PublicationStatus), chaptersCount,
	)
	if err != nil {
		return err
	}

	// Новые главы ищем только у уже известных манг (первичная загрузка без уведомлений).
	// FindNewChaptersSince отсекает старую историю, впервые попавшую в список
	// драйвера (догрузка глав в mgeko): после перехода с 50 карточных глав на
	// полный список подписчикам не уйдут десятки «новых» глав давних месяцев.
	if found {
		newChapters := parsers.FindNewChaptersSince(chapters, state.LatestChapter, state.ChaptersCount)
		if len(newChapters) > 0 {
			c.notify.NewChapters(mangaID, rawURL, card.Title, newChapters)
		}
	}

	return nil
}
