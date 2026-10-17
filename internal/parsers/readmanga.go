package parsers

import (
	"fmt"
	"log"
	"time"

	"github.com/SemenovDmitry/manga-crawler-backend/db"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/telegram"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/utils"
)

// ParseSource парсит мангу для конкретного источника.
// Для каждой манги из списка:
// 1. Ищет RSS ссылку на странице манги
// 2. Получает RSS фид с главами
// 3. Сохраняет новые главы в БД
// 4. Отправляет уведомления в Telegram подписчикам
// Между запросами делает паузу 4 секунды для избежания бана.
func ReadmangaParser(telegramBot *telegram.TelegramBot, source types.Source, mangaList []types.Manga) error {
	log.Printf("Начало проверки обновлений для источника: %s\n\n", source.ParserName)

	for index, manga := range mangaList {
		log.Printf("Проверяем мангу: %s (ID: %d)", manga.Title, manga.ID)

		// Ищем RSS ссылку на странице манги
		rssUrl, err := utils.FindRSSLink(source.BaseURL, manga.URL)
		if err != nil {
			log.Printf("Ошибка поиска RSS ссылки для %s: %v", manga.Title, err)
			telegram.SendErrorNotification(telegramBot, manga.Title)
			continue
		}

		// Получаем RSS фид
		feed, err := utils.GetRSSFeed(rssUrl)
		if err != nil {
			log.Printf("Ошибка получения RSS для %s: %v", manga.Title, err)
			telegram.SendErrorNotification(telegramBot, manga.Title)
			continue
		}

		if len(feed.Items) == 0 {
			log.Printf("Нет глав для %s", manga.Title)
			db.UpdateMangaLastCheck(manga.ID)
			continue
		}

		// Преобразуем RSS в нашу структуру
		transformedFeed, err := utils.TransformRSSFeed(feed)
		if err != nil {
			log.Printf("Ошибка преобразования RSS для %s: %v", manga.Title, err)
			continue
		}

		// Сохраняем новые главы в БД и получаем только реально новые
		newChapters, err := db.CreateChapters(manga.ID, transformedFeed.Chapters)
		if err != nil {
			log.Printf("Ошибка сохранения глав для %s: %v", manga.Title, err)
			continue
		}

		fmt.Printf("Новых глав: %d\n\n", len(newChapters))

		// Если есть новые главы — обновляем последнюю главу и отправляем уведомления
		if len(newChapters) > 0 {
			// Обновляем информацию о последней главе
			lastChapter := newChapters[0]
			if err := db.UpdateMangaLastChapter(manga.ID, lastChapter.URL, lastChapter.Title); err != nil {
				log.Printf("Ошибка обновления последней главы: %v", err)
			}

			// Получаем подписчиков манги
			subscribers, err := db.GetMangaSubscribers(manga.ID)
			if err != nil {
				log.Printf("Ошибка получения подписчиков: %v", err)
			}

			// Отправляем уведомления всем подписчикам
			for _, subscriber := range subscribers {
				telegram.SendMangaUpdateToUser(telegramBot, subscriber.ID, source.BaseURL, manga, newChapters)
			}
		} else {
			// Просто обновляем время последней проверки
			db.UpdateMangaLastCheck(manga.ID)
		}

		// Ждём 4 секунды перед следующей мангой (чтобы не получить бан)
		if index < len(mangaList)-1 {
			fmt.Println("Ожидание 4 секунды...")
			time.Sleep(4 * time.Second)
		}
	}

	log.Println("Проверка обновлений завершена")
	return nil
}
