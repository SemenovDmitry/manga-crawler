package telegram

// HTML-тексты сообщений бота (разметка Telegram parse_mode=HTML).

const TextStart = `
	👋 <b>Привет! Я бот для отслеживания манги.</b>

	Я умею:
	• Отслеживать обновления манги
	• Уведомлять о новых главах

	<b>Команды:</b>
	/help — справка
`

const TextHelp = `
	📖 <b>Справка по командам:</b>

	<b>Команды:</b>
	/start — начать работу
	/help — эта справка
`

const TextUnknownCommand = "Неизвестная команда. Используйте /help для справки."

const TextStartupNotification = `
	🚀 <b>Манга-трекер запущен!</b>
	<i>Начинаю отслеживание обновлений...</i>
`
