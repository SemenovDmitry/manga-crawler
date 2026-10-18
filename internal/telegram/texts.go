package telegram

// HTML-тексты сообщений бота (разметка Telegram parse_mode=HTML).

const textStart = `
	👋 <b>Привет! Я бот для отслеживания манги.</b>

	Я умею:
	• Отслеживать обновления манги
	• Уведомлять о новых главах

	<b>Команды:</b>
	/help — справка
`

const textHelp = `
	📖 <b>Справка по командам:</b>

	<b>Команды:</b>
	/start — начать работу
	/help — эта справка
`

const textUnknownCommand = "Неизвестная команда. Используйте /help для справки."

const textStartupNotification = `
	🚀 <b>Манга-трекер запущен!</b>
	<i>Начинаю отслеживание обновлений...</i>
`
