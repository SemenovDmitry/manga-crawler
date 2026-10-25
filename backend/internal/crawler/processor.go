package crawler

import (
	"context"
	"errors"
	"fmt"
	"log"

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
// карточек логируются и проход не прерывают. Между обработками выдерживается
// пауза вежливости (pacer); label — имя канала для логов прогресса.
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

		if err := c.processManga(ctx, src, rawURL); err != nil {
			log.Printf("[%s] Ошибка обработки %s: %v", src.source.Name, rawURL, err)
		} else {
			res.processed++
		}

		if err := c.pace.wait(ctx); err != nil {
			return res, err
		}
		if (i+1)%100 == 0 {
			log.Printf("[%s] %s: %d/%d", src.source.Name, label, i+1, len(urls))
		}
	}

	return res, nil
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

	body, err := c.fetch.Card(ctx, rawURL)
	if err != nil {
		// Тайтл переехал на зеркало: редирект не переходим, карточка
		// пропускается — она уже учтена по новому адресу, если тот
		// встретится в sitemap или каталоге.
		if errors.Is(err, ErrRedirected) {
			log.Printf("[%s] Пропущен кросс-доменный редирект: %v", src.source.Name, err)
			return nil
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
		// чтобы диагностировать причину.
		c.badPages.dump(src.source, urlPart, body, err)
		return fmt.Errorf("ошибка парсинга страницы: %w", err)
	}

	chapters := parsers.NormalizeChapters(card.Chapters)
	latestChapter := ""
	if len(chapters) > 0 {
		latestChapter = chapters[len(chapters)-1].Number
	}

	mangaID, err := c.store.UpsertManga(
		src.source.ID, urlPart, card.Title, card.AltTitles, latestChapter,
		storage.PublicationStatus(card.PublicationStatus), len(chapters),
	)
	if err != nil {
		return err
	}

	// Новые главы ищем только у уже известных манг (первичная загрузка без уведомлений)
	if found {
		newChapters := parsers.FindNewChapters(chapters, state.ChaptersCount)
		if len(newChapters) > 0 {
			c.notify.NewChapters(mangaID, rawURL, card.Title, newChapters)
		}
	}

	return nil
}
