package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// Тесты handler-логики бота через интерфейс Backend (второй adapter seam-а:
// хранилище в памяти) и через mock-adapter Telegram API. Сеть и Postgres
// в тестах не участвуют; asserts — по наблюдаемым эффектам через seam,
// не по внутреннему состоянию.

// ——— фейк хранилища ———

// fakeBackend — Backend в памяти. Незаглушенные методы встроенного
// nil-интерфейса паникуют громко: тест упадёт, если бот начнёт звать лишнее.
type fakeBackend struct {
	Backend
	users       map[int64]storage.ChatUser
	manga       map[int64]storage.Manga
	subs        map[string]bool // "chatID:mangaID"
	pending     []storage.Notification
	unavailable []int64
	markSent    [][]int64
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		users: map[int64]storage.ChatUser{},
		manga: map[int64]storage.Manga{},
		subs:  map[string]bool{},
	}
}

func (f *fakeBackend) CreateChatUser(user storage.ChatUser) (storage.ChatUser, error) {
	f.users[user.ChatID] = user
	return user, nil
}

func (f *fakeBackend) GetChatUser(chatID int64) (storage.ChatUser, error) {
	user, ok := f.users[chatID]
	if !ok {
		return storage.ChatUser{}, storage.ErrChatUserNotFound
	}
	return user, nil
}

func (f *fakeBackend) MarkChatUserUnavailable(chatID int64) error {
	f.unavailable = append(f.unavailable, chatID)
	return nil
}

func (f *fakeBackend) SearchManga(query string, limit int) ([]storage.Manga, error) {
	var found []storage.Manga
	for _, m := range f.manga {
		if strings.Contains(strings.ToLower(m.Title), strings.ToLower(query)) {
			found = append(found, m)
			if len(found) == limit {
				break
			}
		}
	}
	return found, nil
}

func (f *fakeBackend) GetManga(id int64) (storage.Manga, error) {
	m, ok := f.manga[id]
	if !ok {
		return storage.Manga{}, storage.ErrMangaNotFound
	}
	return m, nil
}

func (f *fakeBackend) Subscribe(chatID, mangaID int64) (bool, error) {
	if f.subs[subKey(chatID, mangaID)] {
		return false, nil
	}
	if _, ok := f.manga[mangaID]; !ok {
		return false, storage.ErrMangaNotFound
	}
	f.subs[subKey(chatID, mangaID)] = true
	return true, nil
}

func (f *fakeBackend) Unsubscribe(chatID, mangaID int64) (bool, error) {
	if !f.subs[subKey(chatID, mangaID)] {
		return false, nil
	}
	delete(f.subs, subKey(chatID, mangaID))
	return true, nil
}

func (f *fakeBackend) UserSubscriptions(chatID int64) ([]storage.Manga, error) {
	var result []storage.Manga
	for _, m := range f.manga {
		if f.subs[subKey(chatID, m.ID)] {
			result = append(result, m)
		}
	}
	return result, nil
}

func (f *fakeBackend) PendingNotifications(limit int) ([]storage.Notification, error) {
	n := min(len(f.pending), limit)
	return f.pending[:n], nil
}

func (f *fakeBackend) MarkNotificationsSent(ids []int64) error {
	f.markSent = append(f.markSent, ids)

	var rest []storage.Notification
	for _, p := range f.pending {
		sent := false
		for _, id := range ids {
			if p.ID == id {
				sent = true
				break
			}
		}
		if !sent {
			rest = append(rest, p)
		}
	}
	f.pending = rest
	return nil
}

func subKey(chatID, mangaID int64) string {
	return fmt.Sprintf("%d:%d", chatID, mangaID)
}

// ——— mock Telegram API ———

// fakeTelegram — Telegram Bot API в памяти: записывает отправленные
// сообщения, чатам из blocked отвечает «бот заблокирован» (403),
// чатам из failing — серверной ошибкой (500). onSend — колбэк после
// каждой отправки (для эмуляции сигнала остановки посреди порции).
type fakeTelegram struct {
	blocked map[int64]bool
	failing map[int64]bool
	onSend  func()
	sent    []sentMessage
}

type sentMessage struct {
	method string
	chatID string
	text   string
}

func (f *fakeTelegram) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	method := req.URL.Path
	if i := strings.LastIndex(method, "/"); i >= 0 {
		method = method[i+1:]
	}

	var reqBody struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	_ = json.Unmarshal(body, &reqBody)

	if reqBody.ChatID != "" {
		chatID, _ := strconv.ParseInt(reqBody.ChatID, 10, 64)
		if method == "sendMessage" || method == "editMessageText" {
			if f.blocked[chatID] {
				return apiResponse(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`), nil
			}
			if f.failing[chatID] {
				return apiResponse(`{"ok":false,"error_code":500,"description":"Internal server error"}`), nil
			}
		}
	}

	if method == "sendMessage" || method == "editMessageText" {
		f.sent = append(f.sent, sentMessage{method: method, chatID: reqBody.ChatID, text: reqBody.Text})
		if f.onSend != nil {
			f.onSend()
		}
	}

	return apiResponse(`{"ok":true}`), nil
}

func apiResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

// newTestBot собирает бота над фейками — без сетевых побочных эффектов
// createBot (меню команд, стартовое уведомление).
func newTestBot(backend Backend, api *fakeTelegram) *Bot {
	return &Bot{
		token:       "test-token",
		ownerChatID: "42",
		miniAppURL:  "",
		client:      api,
		botURL:      "https://api.telegram.local/bottest-token",
		backend:     backend,
	}
}

// chatMessage — сообщение от чата с текстом (From заполнен, как в
// реальных апдейтах: апдейты без From бот пропускает).
func chatMessage(chatID int64, text string) Update {
	return Update{Message: &Message{Chat: &Chat{ID: chatID}, From: &User{ID: chatID}, Text: text}}
}

// callbackTap — нажатие inline-кнопки follow/unfollow в чате.
func callbackTap(chatID, messageID, mangaID int64, prefix string) *CallbackQuery {
	return &CallbackQuery{
		ID: "cb-1",
		Message: &Message{Chat: &Chat{ID: chatID}, MessageID: messageID},
		Data:    fmt.Sprintf("%s:%d", prefix, mangaID),
	}
}

// lastSent возвращает последнее отправленное сообщение.
func lastSent(t *testing.T, api *fakeTelegram) sentMessage {
	t.Helper()
	if len(api.sent) == 0 {
		t.Fatal("ни одного сообщения не отправлено")
	}
	return api.sent[len(api.sent)-1]
}

// ——— команды ———

func TestStartSavesChatUserAndGreets(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/start"))

	if _, ok := fb.users[100]; !ok {
		t.Error("пользователь чата не зарегистрирован при /start")
	}
	sent := lastSent(t, api)
	if sent.text != textStart {
		t.Errorf("ответ на /start: %q, ожидался textStart", sent.text)
	}
}

func TestHelpAnswersWithoutBackendCalls(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/help"))

	if len(fb.users) != 0 {
		t.Error("/help не должен регистрировать пользователя")
	}
	if sent := lastSent(t, api); sent.text != textHelp {
		t.Errorf("ответ на /help: %q, ожидался textHelp", sent.text)
	}
}

func TestUnknownCommand(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/subscribe"))

	if sent := lastSent(t, api); sent.text != textUnknownCommand {
		t.Errorf("ответ на неизвестную команду: %q", sent.text)
	}
}

func TestFindShowsResultsWithFollowButtons(t *testing.T) {
	fb := newFakeBackend()
	fb.manga[7] = storage.Manga{ID: 7, Title: "Shadow Slave", BaseURL: "https://mgeko.cc", URL: "manga/j3pu-shadow-slave", ChaptersCount: 120}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/find shadow"))

	sent := lastSent(t, api)
	for _, want := range []string{"Shadow Slave", "mgeko.cc", "120"} {
		if !strings.Contains(sent.text, want) {
			t.Errorf("в выдаче поиска нет %q:\n%s", want, sent.text)
		}
	}
	// Кнопки follow с callback-данными проверяются в mangaView_test.go
	// (TestFindResultsKeyboard): тот же построитель клавиатуры.
}

func TestFindEmpty(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/find неттакой"))

	if sent := lastSent(t, api); sent.text != fmt.Sprintf(textNothingFound, "неттакой") {
		t.Errorf("ответ на пустой поиск: %q", sent.text)
	}
}

func TestFindWithoutQueryShowsUsage(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/find"))

	if sent := lastSent(t, api); sent.text != textFindUsage {
		t.Errorf("ответ на /find без запроса: %q", sent.text)
	}
}

func TestListShowsSubscriptions(t *testing.T) {
	fb := newFakeBackend()
	fb.manga[3] = storage.Manga{ID: 3, Title: "Solo Leveling", BaseURL: "https://1.seimanga.me", URL: "solo_leveling"}
	fb.subs["100:3"] = true
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/list"))

	sent := lastSent(t, api)
	if !strings.Contains(sent.text, "Solo Leveling") {
		t.Errorf("в /list нет подписки:\n%s", sent.text)
	}
}

func TestListEmpty(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/list"))

	if sent := lastSent(t, api); sent.text != textEmptyList {
		t.Errorf("ответ на пустой /list: %q", sent.text)
	}
}

// ——— блокировка чата ———

func TestBlockedChatGetsOnlyAccessDenied(t *testing.T) {
	fb := newFakeBackend()
	fb.users[100] = storage.ChatUser{ChatID: 100, Blocked: true}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/start"))

	if sent := lastSent(t, api); sent.text != textAccessDenied {
		t.Errorf("заблокированный чат должен получать только отказ: %q", sent.text)
	}
	if len(fb.users) != 1 {
		t.Error("заблокированный чат не должен перерегистрироваться")
	}

	// /find тоже не обслуживается.
	bot.handleChatMessage(chatMessage(100, "/find shadow"))
	if n := len(api.sent); n != 1 {
		t.Errorf("сообщений после /find заблокированного: %d, ожидалось 1", n)
	}
}

// ——— inline-кнопки подписки ———

func TestFollowCallbackSubscribesAndConfirms(t *testing.T) {
	fb := newFakeBackend()
	fb.manga[7] = storage.Manga{ID: 7, Title: "Shadow Slave", BaseURL: "https://mgeko.cc", URL: "manga/j3pu-shadow-slave"}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleCallbackQuery(callbackTap(100, 55, 7, callbackFollow))

	if !fb.subs["100:7"] {
		t.Error("подписка не создана по кнопке follow")
	}
	sent := lastSent(t, api)
	if sent.method != "editMessageText" || !strings.Contains(sent.text, "Shadow Slave") {
		t.Errorf("подтверждение подписки должно прийти правкой сообщения с ссылкой на мангу:\n%s", sent.text)
	}
}

func TestFollowAlreadySubscribed(t *testing.T) {
	fb := newFakeBackend()
	fb.manga[7] = storage.Manga{ID: 7, Title: "Shadow Slave", BaseURL: "https://mgeko.cc", URL: "manga/x"}
	fb.subs["100:7"] = true
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleCallbackQuery(callbackTap(100, 55, 7, callbackFollow))

	if sent := lastSent(t, api); !strings.Contains(sent.text, "уже подписаны") {
		t.Errorf("повторная подписка: %q", sent.text)
	}
}

func TestFollowDeletedMangaReportsNotFound(t *testing.T) {
	// Манга не найдена — «манга не найдена», а не «ошибка сервера».
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleCallbackQuery(callbackTap(100, 55, 999, callbackFollow))

	if sent := lastSent(t, api); !strings.Contains(sent.text, "не найдена") {
		t.Errorf("подписка на несуществующую мангу: %q", sent.text)
	}
	if len(fb.subs) != 0 {
		t.Error("подписки на несуществующую мангу быть не должно")
	}
}

func TestUnsubscribeCallbackFlows(t *testing.T) {
	fb := newFakeBackend()
	fb.manga[7] = storage.Manga{ID: 7, Title: "Shadow Slave", BaseURL: "https://mgeko.cc", URL: "manga/x"}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	// Отписка без подписки — «не были подписаны».
	bot.handleCallbackQuery(callbackTap(100, 55, 7, callbackUnfollow))
	if sent := lastSent(t, api); !strings.Contains(sent.text, "не были подписаны") {
		t.Errorf("отписка без подписки: %q", sent.text)
	}

	// Отписка от существующей подписки.
	fb.subs["100:7"] = true
	bot.handleCallbackQuery(callbackTap(100, 55, 7, callbackUnfollow))
	if fb.subs["100:7"] {
		t.Error("подписка не снята")
	}
	if sent := lastSent(t, api); !strings.Contains(sent.text, "Подписка отменена") {
		t.Errorf("отписка: %q", sent.text)
	}
}

func TestCallbackUnknownAction(t *testing.T) {
	fb := newFakeBackend()
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleCallbackQuery(&CallbackQuery{ID: "cb", Message: &Message{Chat: &Chat{ID: 100}}, Data: "hack:1"})
	// Никаких подписок/сообщений — только ответ на кнопку.
	if len(api.sent) != 0 {
		t.Errorf("неизвестное действие не должно отправлять сообщений: %v", api.sent)
	}
}

// ——— рассылка уведомлений (модуль очереди) ———

func TestDispatchBatchSendsAndMarks(t *testing.T) {
	fb := newFakeBackend()
	fb.pending = []storage.Notification{
		{ID: 1, ChatID: 100, Text: "новые главы"},
		{ID: 2, ChatID: 200, Text: "тоже новые"},
	}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.dispatchBatch(context.Background())

	if n := len(api.sent); n != 2 {
		t.Fatalf("отправлено сообщений: %d, ожидалось 2", n)
	}
	// Оба отмечены отправленными, очередь пуста.
	if len(fb.markSent) != 1 || len(fb.markSent[0]) != 2 {
		t.Errorf("отметки отправленных: %v", fb.markSent)
	}
	if len(fb.pending) != 0 {
		t.Errorf("очередь не опустела: %v", fb.pending)
	}
	if len(fb.unavailable) != 0 {
		t.Errorf("неожиданные снятия чатов: %v", fb.unavailable)
	}
}

func TestDispatchBatchBlockedChatMarkedUnavailable(t *testing.T) {
	// Чат 200 заблокировал бота: уведомление снимается с очереди
	// (считается доставленным), чат помечается недоступным.
	fb := newFakeBackend()
	fb.pending = []storage.Notification{
		{ID: 1, ChatID: 100, Text: "доставится"},
		{ID: 2, ChatID: 200, Text: "заблокирован"},
	}
	api := &fakeTelegram{blocked: map[int64]bool{200: true}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.dispatchBatch(context.Background())

	if len(fb.unavailable) != 1 || fb.unavailable[0] != 200 {
		t.Errorf("недоступные чаты: %v, ожидался [200]", fb.unavailable)
	}
	if len(fb.markSent) != 1 || len(fb.markSent[0]) != 2 {
		t.Errorf("оба уведомления (доставленное и снятое) отмечаются: %v", fb.markSent)
	}
	if len(fb.pending) != 0 {
		t.Errorf("снятое уведомление должно покинуть очередь: %v", fb.pending)
	}
}

func TestDispatchBatchKeepsFailedNotifications(t *testing.T) {
	// Сбой доставки (не блокировка) — уведомление остаётся в очереди
	// и не помечается отправленным.
	fb := newFakeBackend()
	fb.pending = []storage.Notification{
		{ID: 1, ChatID: 300, Text: "упадёт"},
	}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{300: true}}
	bot := newTestBot(fb, api)

	bot.dispatchBatch(context.Background())

	if len(fb.pending) != 1 || fb.pending[0].ID != 1 {
		t.Errorf("несостоявшееся уведомление должно остаться в очереди: %v", fb.pending)
	}
	for _, ids := range fb.markSent {
		for _, id := range ids {
			if id == 1 {
				t.Error("несостоявшееся уведомление отмечено отправленным")
			}
		}
	}
}

func TestDispatchBatchStopsOnContextCancel(t *testing.T) {
	// Сигнал остановки пришёл посреди порции (после первого сообщения):
	// отправленное отмечается, недоставленное остаётся в очереди.
	fb := newFakeBackend()
	fb.pending = []storage.Notification{
		{ID: 1, ChatID: 100, Text: "первое"},
		{ID: 2, ChatID: 100, Text: "второе"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := &fakeTelegram{
		blocked: map[int64]bool{},
		failing: map[int64]bool{},
		onSend:  cancel,
	}
	bot := newTestBot(fb, api)

	bot.dispatchBatch(ctx)

	if len(api.sent) != 1 {
		t.Errorf("после отмены должно уйти ровно одно сообщение: %d", len(api.sent))
	}
	if len(fb.markSent) != 1 || len(fb.markSent[0]) != 1 || fb.markSent[0][0] != 1 {
		t.Errorf("отмечено отправленным только первое: %v", fb.markSent)
	}
	if len(fb.pending) != 1 || fb.pending[0].ID != 2 {
		t.Errorf("второе уведомление должно остаться в очереди: %v", fb.pending)
	}
}

// ——— доступ: fail-closed при ошибке БД ———

func TestUserBlockedFailsClosedOnDBError(t *testing.T) {
	// Ошибка чтения чата — не обслуживаем: сообщения не уходят.
	fb := &failingUsersBackend{Backend: newFakeBackend()}
	api := &fakeTelegram{blocked: map[int64]bool{}, failing: map[int64]bool{}}
	bot := newTestBot(fb, api)

	bot.handleChatMessage(chatMessage(100, "/find shadow"))

	if len(api.sent) != 0 {
		t.Errorf("при недоступной БД сообщения не должны уходить: %v", api.sent)
	}
}

type failingUsersBackend struct {
	Backend
}

func (f *failingUsersBackend) GetChatUser(int64) (storage.ChatUser, error) {
	return storage.ChatUser{}, errors.New("бд недоступна")
}
