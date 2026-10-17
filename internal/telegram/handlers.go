package telegram

import (
	"fmt"
	"log"
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/db"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/utils"
)

// HandleMessage обрабатывает входящее сообщение
func HandleMessage(bot *TelegramBot, msg *TgMessage) {
	if msg == nil || msg.From == nil {
		return
	}

	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	// Регистрируем/обновляем пользователя
	_, err := db.GetOrCreateUser(
		msg.From.ID,
		msg.From.Username,
		msg.From.FirstName,
		msg.From.LastName,
	)
	if err != nil {
		log.Printf("Ошибка регистрации пользователя: %v", err)
	}

	// Обрабатываем команды
	switch {
	case text == "/start":
		handleStart(bot, chatID)
	case text == "/help":
		handleHelp(bot, chatID)
	case text == "/sources":
		handleSources(bot, chatID)
	case text == "/list":
		handleList(bot, chatID, msg.From.ID)
	case strings.HasPrefix(text, "/add"):
		handleAdd(bot, chatID, msg.From.ID, text)
	case strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://"):
		// Если пользователь просто отправил URL
		handleAddManga(bot, chatID, msg.From.ID, text)
	default:
		// Неизвестная команда
		sendMessageToChat(bot, chatID, "Неизвестная команда. Используйте /help для справки.")
	}
}

// handleStart обработка команды /start
func handleStart(bot *TelegramBot, chatID int64) {
	message := `👋 <b>Привет! Я бот для отслеживания манги.</b>

Я умею:
• Отслеживать обновления манги
• Уведомлять о новых главах

<b>Как добавить мангу:</b>
Просто отправь мне ссылку на мангу, например:
<code>https://a.zazaza.me/asura_2024</code>

<b>Команды:</b>
/sources — список источников
/add — добавить мангу
/list — мои подписки
/help — справка`

	sendMessageToChat(bot, chatID, message)
}

// handleHelp обработка команды /help
func handleHelp(bot *TelegramBot, chatID int64) {
	message := `📖 <b>Справка по командам:</b>

<b>Добавление манги:</b>
Отправь ссылку на мангу:
<code>https://a.zazaza.me/название_манги</code>

<b>Команды:</b>
/start — начать работу
/sources — список поддерживаемых источников
/add — добавить мангу (ожидает URL)
/list — список отслеживаемых манг
/help — эта справка`

	sendMessageToChat(bot, chatID, message)
}

// handleSources обработка команды /sources
func handleSources(bot *TelegramBot, chatID int64) {
	sources, err := db.GetActiveSources()
	if err != nil {
		log.Printf("Ошибка получения источников: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка получения списка источников")
		return
	}

	if len(sources) == 0 {
		sendMessageToChat(bot, chatID, "📭 Нет доступных источников")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌐 <b>Поддерживаемые источники (%d):</b>\n\n", len(sources)))

	for i, source := range sources {
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, source.ParserName))
		sb.WriteString(fmt.Sprintf("   🔗 %s\n\n", source.BaseURL))
	}

	sb.WriteString("<b>Как добавить мангу:</b>\n")
	sb.WriteString("Скопируйте URL страницы манги и отправьте его боту.\n")
	sb.WriteString("Или используйте команду /add URL")

	sendMessageToChat(bot, chatID, sb.String())
}

// handleList обработка команды /list
func handleList(bot *TelegramBot, chatID int64, userID int64) {
	mangaList, err := db.GetUserSubscriptions(userID)
	if err != nil {
		log.Printf("Ошибка получения подписок: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка получения списка манги")
		return
	}

	if len(mangaList) == 0 {
		sendMessageToChat(bot, chatID, "📭 У вас нет отслеживаемых манг.\n\nОтправьте ссылку на мангу, чтобы добавить её.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 <b>Ваши манги (%d):</b>\n", len(mangaList)))

	// Группируем манги по источнику
	grouped := make(map[string][]db.MangaWithSource)
	var sourceOrder []string

	for _, manga := range mangaList {
		if _, exists := grouped[manga.SourceName]; !exists {
			sourceOrder = append(sourceOrder, manga.SourceName)
		}
		grouped[manga.SourceName] = append(grouped[manga.SourceName], manga)
	}

	// Выводим по источникам
	for _, sourceName := range sourceOrder {
		mangas := grouped[sourceName]
		sb.WriteString(fmt.Sprintf("\n🌐 <b>%s</b> (%d):\n", escapeHTML(sourceName), len(mangas)))
		for i, manga := range mangas {
			mangaURL := fmt.Sprintf("%s/%s", manga.SourceBaseURL, manga.URL)
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a>\n", i+1, mangaURL, escapeHTML(manga.Title)))
		}
	}

	sendMessageToChat(bot, chatID, sb.String())
}

// handleAdd обработка команды /add
func handleAdd(bot *TelegramBot, chatID int64, userID int64, text string) {
	// Убираем /add и пробелы
	url := strings.TrimSpace(strings.TrimPrefix(text, "/add"))

	if url == "" {
		sendMessageToChat(bot, chatID, "❓ Отправьте ссылку на мангу после команды /add или просто отправьте ссылку.")
		return
	}

	handleAddManga(bot, chatID, userID, url)
}

// handleAddManga обработка добавления манги по URL
func handleAddManga(bot *TelegramBot, chatID int64, userID int64, rawURL string) {
	// Парсим URL
	parsed, err := utils.ParseMangaURL(rawURL)
	if err != nil {
		sendMessageToChat(bot, chatID, fmt.Sprintf("❌ Некорректный URL: %v", err))
		return
	}

	// Ищем источник по базовому URL
	source, err := db.GetSourceByBaseURL(parsed.BaseURL)
	if err != nil {
		log.Printf("Ошибка поиска источника: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка при поиске источника")
		return
	}

	if source == nil {
		// Пробуем поиск по хосту
		source, err = db.GetSourceByBaseURL(parsed.Host)
		if err != nil {
			log.Printf("Ошибка поиска источника по хосту: %v", err)
			sendMessageToChat(bot, chatID, "❌ Ошибка при поиске источника")
			return
		}
	}

	if source == nil {
		sendMessageToChat(bot, chatID, fmt.Sprintf("❌ Источник <b>%s</b> не поддерживается.\n\nПоддерживаемые сайты:\n• readmanga (a.zazaza.me)\n• mintmanga (1.seimanga.me)", parsed.Host))
		return
	}

	// Проверяем, есть ли уже такая манга в БД
	existingManga, err := db.GetMangaBySourceAndURL(source.ID, parsed.MangaPath)
	if err != nil {
		log.Printf("Ошибка проверки манги: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка при проверке манги")
		return
	}

	if existingManga != nil {
		// Манга уже существует — проверяем подписку
		subscription, err := db.GetSubscription(userID, existingManga.ID)
		if err != nil {
			log.Printf("Ошибка проверки подписки: %v", err)
		}

		if subscription != nil {
			// Уже подписан
			chapters, _ := db.GetChaptersByMangaID(existingManga.ID)
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("ℹ️ Вы уже отслеживаете <b>%s</b>\n\n", escapeHTML(existingManga.Title)))

			if existingManga.LastChapterTitle != "" {
				sb.WriteString(fmt.Sprintf("📖 Последняя глава: %s\n", escapeHTML(existingManga.LastChapterTitle)))
			}

			sb.WriteString(fmt.Sprintf("📚 Всего глав: %d", len(chapters)))

			sendMessageToChat(bot, chatID, sb.String())
			return
		}

		// Не подписан — создаём подписку
		_, err = db.CreateSubscription(userID, existingManga.ID)
		if err != nil {
			log.Printf("Ошибка создания подписки: %v", err)
			sendMessageToChat(bot, chatID, "❌ Ошибка при создании подписки")
			return
		}

		chapters, _ := db.GetChaptersByMangaID(existingManga.ID)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("✅ Вы подписались на <b>%s</b>\n\n", escapeHTML(existingManga.Title)))

		if existingManga.LastChapterTitle != "" {
			sb.WriteString(fmt.Sprintf("📖 Последняя глава: %s\n", escapeHTML(existingManga.LastChapterTitle)))
		}

		sb.WriteString(fmt.Sprintf("📚 Всего глав: %d\n\n", len(chapters)))
		sb.WriteString("Теперь вы будете получать уведомления о новых главах!")

		sendMessageToChat(bot, chatID, sb.String())
		return
	}

	// Манга не существует — пробуем получить данные
	sendMessageToChat(bot, chatID, "🔍 Ищу мангу...")

	// Пробуем найти RSS и получить информацию о манге
	rssURL, err := utils.FindRSSLink(source.BaseURL, parsed.MangaPath)
	if err != nil {
		log.Printf("Ошибка поиска RSS: %v", err)
		sendMessageToChat(bot, chatID, fmt.Sprintf("❌ Не удалось найти мангу по адресу:\n%s\n\nПроверьте URL и попробуйте снова.", rawURL))
		return
	}

	feed, err := utils.GetRSSFeed(rssURL)
	if err != nil {
		log.Printf("Ошибка получения RSS: %v", err)
		sendMessageToChat(bot, chatID, "❌ Не удалось получить данные о манге")
		return
	}

	if len(feed.Items) == 0 {
		sendMessageToChat(bot, chatID, "❌ Манга найдена, но глав пока нет")
		return
	}

	// Преобразуем и получаем название
	transformedFeed, err := utils.TransformRSSFeed(feed)
	if err != nil {
		log.Printf("Ошибка преобразования RSS: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка обработки данных манги")
		return
	}

	// Создаём мангу в БД
	newManga, err := db.CreateManga(source.ID, parsed.MangaPath, transformedFeed.Title)
	if err != nil {
		log.Printf("Ошибка создания манги: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка при сохранении манги")
		return
	}

	// Сохраняем главы
	_, err = db.CreateChapters(newManga.ID, transformedFeed.Chapters)
	if err != nil {
		log.Printf("Ошибка сохранения глав: %v", err)
	}

	// Обновляем последнюю главу
	if len(transformedFeed.Chapters) > 0 {
		lastChapter := transformedFeed.Chapters[0]
		db.UpdateMangaLastChapter(newManga.ID, lastChapter.URL, lastChapter.Title)
	}

	// Создаём подписку
	_, err = db.CreateSubscription(userID, newManga.ID)
	if err != nil {
		log.Printf("Ошибка создания подписки: %v", err)
		sendMessageToChat(bot, chatID, "❌ Ошибка при создании подписки")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Манга <b>%s</b> добавлена!\n\n", escapeHTML(transformedFeed.Title)))
	sb.WriteString(fmt.Sprintf("📚 Найдено глав: %d\n", len(transformedFeed.Chapters)))

	if len(transformedFeed.Chapters) > 0 {
		sb.WriteString(fmt.Sprintf("📖 Последняя глава: %s\n\n", escapeHTML(transformedFeed.Chapters[0].Title)))
	}

	sb.WriteString("Теперь вы будете получать уведомления о новых главах!")

	sendMessageToChat(bot, chatID, sb.String())
}

// sendMessageToChat отправляет сообщение в указанный чат
func sendMessageToChat(bot *TelegramBot, chatID int64, text string) error {
	return sendMessageToUser(bot, chatID, text)
}
