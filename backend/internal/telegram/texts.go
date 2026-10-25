package telegram

// HTML-тексты сообщений бота (разметка Telegram parse_mode=HTML).

const textStart = `
	👋 <b>Привет! Я бот для отслеживания манги.</b>

	<p>
		Подпишись на мангу, 
		проверка происходит каждые 30 минут 
		и вам придет уведомление.
	</p>

	<b>Команды:</b>
	/find название — поиск манги (подписка в один тап по кнопке)
	/list — мои подписки (отписка по кнопке)
	/help — справка
`

const textHelp = `
	📖 <b>Справка по командам:</b>

	/find название — поиск манги по названию
	После поиска можно подписаться кнопкой под результатами.
	/list — список ваших подписок (тоже с кнопками отписки)

	Ищу мангу и уведомляю о новых главах на сайтах:
	• 1.seimanga.me
	• a.zazaza.me
	• mgeko.cc

	Новые главы приходят автоматически после каждого обновления каталога.
`

const textUnknownCommand = "Неизвестная команда. Используйте /help для справки."

const textStartupNotification = `
	🚀 <b>Манга-трекер запущен!</b>
	<i>Начинаю отслеживание обновлений...</i>
`

const textFindUsage = "🔍 Укажите название: /find <название>"

const textNothingFound = "😕 По запросу «%s» ничего не найдено."

const textSearchError = "😔 Ошибка поиска, попробуйте позже."

const textServerError = "😔 Ошибка сервера, попробуйте позже."

const textFindButtonsHint = "Нажмите на название ниже, чтобы подписаться 👇"

const textButtonExpired = "Сообщение устарело, отправьте /find ещё раз"

const textUnknownAction = "Неизвестное действие"

const textFollowed = "✅ Вы подписаны на: %s\nНовые главы буду присылать сюда."

const textAlreadyFollowed = "Вы уже подписаны на: %s"

const textUnfollowed = "🔕 Подписка отменена: %s"

const textNotFollowed = "Вы не были подписаны на: %s"

const textMangaNotFound = "😕 Манга с ID %d не найдена. Найдите её через /find <название>"

const textEmptyList = "📭 У вас нет подписок. Найдите мангу: /find <название>"

const textAccessDenied = "🚫 Ваш доступ к боту заблокирован."
