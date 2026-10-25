package adminapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// fakeStore — in-memory подмена storage.Store для httptest-тестов.
type fakeStore struct {
	mu          sync.Mutex
	sources     map[int]storage.Source
	chatusers   map[int64]storage.ChatUser
	manga       map[int64]storage.Manga
	nextMangaID int64
	sessions    map[string]storage.AdminSession // ключ — хэш refresh-токена
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sources:   map[int]storage.Source{1: {ID: 1, Name: "grouple", Engine: "grouple", Enabled: true, FullCrawlIntervalSecs: 86400}},
		chatusers: map[int64]storage.ChatUser{100: {ChatID: 100, Username: "reader", IsActive: true}},
		manga: map[int64]storage.Manga{
			42: {ID: 42, SourceID: 1, SourceName: "grouple", URL: "asura_2024", Title: "Одна пьеса",
				AltTitles: "One Piece", LatestChapter: "1120", ChaptersCount: 1120, PublicationStatus: storage.StatusOngoing},
		},
		nextMangaID: 43,
		sessions:    map[string]storage.AdminSession{},
	}
}

func (f *fakeStore) GetAdminSession(tokenHash string) (storage.AdminSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sess, ok := f.sessions[tokenHash]
	if !ok {
		return storage.AdminSession{}, storage.ErrSessionNotFound
	}
	return sess, nil
}

func (f *fakeStore) RotateAdminSession(oldHash string, new storage.AdminSession) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	old, ok := f.sessions[oldHash]
	if !ok || old.Revoked || !time.Now().Before(old.ExpiresAt) {
		return false, nil
	}
	old.Revoked = true
	f.sessions[oldHash] = old
	f.sessions[new.TokenHash] = new
	return true, nil
}

func (f *fakeStore) RevokeAdminSession(tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sess, ok := f.sessions[tokenHash]; ok {
		sess.Revoked = true
		f.sessions[tokenHash] = sess
	}
	return nil
}

func (f *fakeStore) CreateAdminSession(sess storage.AdminSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[sess.TokenHash] = sess
	return nil
}

func (f *fakeStore) GetSources(filter storage.SourceFilter) ([]storage.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []storage.Source
	for _, src := range f.sources {
		if filter.Enabled != nil && src.Enabled != *filter.Enabled {
			continue
		}
		out = append(out, src)
	}
	return out, nil
}

func (f *fakeStore) SetSourceEnabled(id int, enabled bool) (storage.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	src, ok := f.sources[id]
	if !ok {
		return storage.Source{}, storage.ErrSourceNotFound
	}
	src.Enabled = enabled
	f.sources[id] = src
	return src, nil
}

func (f *fakeStore) ResetSourceFullCrawl(id int) (storage.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	src, ok := f.sources[id]
	if !ok {
		return storage.Source{}, storage.ErrSourceNotFound
	}
	src.NextFullCrawlAt = sql.NullTime{}
	f.sources[id] = src
	return src, nil
}

func (f *fakeStore) ListChatUsers(filter storage.ChatUserFilter) ([]storage.ChatUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []storage.ChatUser
	for _, u := range f.chatusers {
		if filter.Active != nil && u.IsActive != *filter.Active {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeStore) SetChatUserBlocked(chatID int64, blocked bool) (storage.ChatUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.chatusers[chatID]
	if !ok {
		return storage.ChatUser{}, storage.ErrChatUserNotFound
	}
	u.Blocked = blocked
	f.chatusers[chatID] = u
	return u, nil
}

func (f *fakeStore) ListManga(filter storage.MangaFilter) ([]storage.Manga, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []storage.Manga
	for _, m := range f.manga {
		if filter.SourceID != 0 && m.SourceID != filter.SourceID {
			continue
		}
		if filter.PublicationStatus != "" && m.PublicationStatus != filter.PublicationStatus {
			continue
		}
		out = append(out, m)
	}
	if filter.Offset > 0 && filter.Offset < len(out) {
		out = out[filter.Offset:]
	} else if filter.Offset > 0 {
		out = nil
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (f *fakeStore) GetManga(id int64) (storage.Manga, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.manga[id]
	if !ok {
		return storage.Manga{}, storage.ErrMangaNotFound
	}
	return m, nil
}

func (f *fakeStore) CreateManga(m storage.Manga) (storage.Manga, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Как FK в реальной БД: несуществующий источник — ErrSourceNotFound.
	if _, ok := f.sources[m.SourceID]; !ok {
		return storage.Manga{}, storage.ErrSourceNotFound
	}
	m.ID = f.nextMangaID
	f.nextMangaID++
	f.manga[m.ID] = m
	return m, nil
}

func (f *fakeStore) UpdateManga(id int64, patch storage.MangaPatch) (storage.Manga, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.manga[id]
	if !ok {
		return storage.Manga{}, storage.ErrMangaNotFound
	}
	if patch.Title != nil {
		m.Title = *patch.Title
	}
	if patch.AltTitles != nil {
		m.AltTitles = *patch.AltTitles
	}
	if patch.LatestChapter != nil {
		m.LatestChapter = *patch.LatestChapter
	}
	if patch.ChaptersCount != nil {
		m.ChaptersCount = *patch.ChaptersCount
	}
	if patch.PublicationStatus != nil {
		if !patch.PublicationStatus.Valid() {
			return storage.Manga{}, fmt.Errorf("CHECK publication_status: %q", *patch.PublicationStatus)
		}
		m.PublicationStatus = *patch.PublicationStatus
	}
	f.manga[id] = m
	return m, nil
}

func (f *fakeStore) DeleteManga(id int64) (storage.Manga, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.manga[id]
	if !ok {
		return storage.Manga{}, storage.ErrMangaNotFound
	}
	delete(f.manga, id)
	return m, nil
}

// --- тесты ---

func testServer(t *testing.T) (*gin.Engine, *fakeStore) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	cfg := testConfig()
	cfg.PasswordHash = string(hash)
	store := newFakeStore()
	return newEngine(store, cfg), store
}

func doJSON(t *testing.T, r *gin.Engine, method, target, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, r *gin.Engine) (access, refresh string) {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/admin-api/auth/login", "",
		jsonBody(t, map[string]string{"username": "admin", "password": "test-password"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var pair struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := parseJSON(rec.Body.String(), &pair); err != nil {
		t.Fatalf("разбор token pair: %v", err)
	}
	return pair.AccessToken, pair.RefreshToken
}

// TestLoginFlow — вход и доступ к защищённому роуту с access-токеном.
func TestLoginFlow(t *testing.T) {
	r, _ := testServer(t)

	// Неверный пароль — 401.
	rec := doJSON(t, r, http.MethodPost, "/admin-api/auth/login", "",
		jsonBody(t, map[string]string{"username": "admin", "password": "wrong"}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("неверный пароль: код %d, ожидался 401", rec.Code)
	}

	// Корректный вход.
	access, _ := login(t, r)

	// Без токена — 401 + WWW-Authenticate.
	rec = doJSON(t, r, http.MethodGet, "/admin-api/sources", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("без токена: код %d, ожидался 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
		t.Errorf("WWW-Authenticate = %q, ожидался Bearer", got)
	}

	// С мусорным токеном — 401.
	rec = doJSON(t, r, http.MethodGet, "/admin-api/sources", "garbage", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("мусорный токен: код %d, ожидался 401", rec.Code)
	}

	// С валидным токеном — 200 и схема ответа.
	rec = doJSON(t, r, http.MethodGet, "/admin-api/sources", access, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("с токеном: код %d, тело %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Sources []struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"sources"`
	}
	if err := parseJSON(rec.Body.String(), &list); err != nil {
		t.Fatalf("разбор sources: %v", err)
	}
	if len(list.Sources) != 1 || list.Sources[0].Name != "grouple" {
		t.Errorf("sources = %+v, ожидался один источник grouple", list.Sources)
	}
}

// TestRefreshRotation — ротация и отказ при повторном использовании refresh.
func TestRefreshRotation(t *testing.T) {
	r, _ := testServer(t)
	_, refresh1 := login(t, r)

	// Первый refresh — успех, приходит новая пара.
	rec := doJSON(t, r, http.MethodPost, "/admin-api/auth/refresh", "",
		jsonBody(t, map[string]string{"refresh_token": refresh1}))
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: код %d, тело %s", rec.Code, rec.Body.String())
	}

	// Повторное использование старого refresh — 401.
	rec = doJSON(t, r, http.MethodPost, "/admin-api/auth/refresh", "",
		jsonBody(t, map[string]string{"refresh_token": refresh1}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("повторный refresh: код %d, ожидался 401", rec.Code)
	}

	// Неизвестный refresh — 401.
	rec = doJSON(t, r, http.MethodPost, "/admin-api/auth/refresh", "",
		jsonBody(t, map[string]string{"refresh_token": "no-such-token"}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("неизвестный refresh: код %d, ожидался 401", rec.Code)
	}
}

// TestLogoutRevokesRefresh — logout гасит refresh-токен.
func TestLogoutRevokesRefresh(t *testing.T) {
	r, _ := testServer(t)
	access, refresh := login(t, r)

	rec := doJSON(t, r, http.MethodPost, "/admin-api/auth/logout", access,
		jsonBody(t, map[string]string{"refresh_token": refresh}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: код %d, тело %s", rec.Code, rec.Body.String())
	}

	// Отозванный refresh больше не работает.
	rec = doJSON(t, r, http.MethodPost, "/admin-api/auth/refresh", "",
		jsonBody(t, map[string]string{"refresh_token": refresh}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh после logout: код %d, ожидался 401", rec.Code)
	}
}

// TestQueryValidation — enum и диапазоны query-параметров.
func TestQueryValidation(t *testing.T) {
	r, _ := testServer(t)
	access, _ := login(t, r)

	cases := []struct {
		query string
		want  int
	}{
		{"?limit=abc", http.StatusBadRequest},              // не парсится — биндинг
		{"?limit=0", http.StatusBadRequest},                // ниже минимума
		{"?limit=201", http.StatusBadRequest},              // выше максимума
		{"?status=bogus", http.StatusBadRequest},           // невалидный enum
		{"?source_id=-1", http.StatusBadRequest},           // отрицательный
		{"?status=completed&limit=100", http.StatusOK},     // валидные
		{"?query=пьеса", http.StatusOK},                    // поиск
	}
	for _, tc := range cases {
		rec := doJSON(t, r, http.MethodGet, "/admin-api/manga"+tc.query, access, "")
		if rec.Code != tc.want {
			t.Errorf("GET /manga%s: код %d, ожидался %d", tc.query, rec.Code, tc.want)
		}
	}
}

// TestPatchMangaPartial — PATCH меняет только переданные поля.
func TestPatchMangaPartial(t *testing.T) {
	r, store := testServer(t)
	access, _ := login(t, r)

	rec := doJSON(t, r, http.MethodPatch, "/admin-api/manga/42", access,
		jsonBody(t, map[string]any{"title": "Новое название"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: код %d, тело %s", rec.Code, rec.Body.String())
	}

	m, err := store.GetManga(42)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Title != "Новое название" {
		t.Errorf("title = %q, ожидалось %q", m.Title, "Новое название")
	}
	if m.AltTitles != "One Piece" || m.LatestChapter != "1120" || m.ChaptersCount != 1120 {
		t.Errorf("неизменённые поля затёрты: %+v", m)
	}

	// PATCH несуществующей — 404.
	rec = doJSON(t, r, http.MethodPatch, "/admin-api/manga/999", access,
		jsonBody(t, map[string]any{"title": "x"}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("patch 999: код %d, ожидался 404", rec.Code)
	}
}

// TestDeleteMangaReturnsBody — DELETE возвращает удалённую запись.
func TestDeleteMangaReturnsBody(t *testing.T) {
	r, store := testServer(t)
	access, _ := login(t, r)

	rec := doJSON(t, r, http.MethodDelete, "/admin-api/manga/42", access, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: код %d, тело %s", rec.Code, rec.Body.String())
	}

	var deleted map[string]any
	if err := parseJSON(rec.Body.String(), &deleted); err != nil {
		t.Fatalf("разбор удалённой записи: %v", err)
	}
	if int(deleted["id"].(float64)) != 42 {
		t.Errorf("id = %v, ожидался 42", deleted["id"])
	}
	if deleted["title"] != "Одна пьеса" {
		t.Errorf("title = %v, ожидалось %q", deleted["title"], "Одна пьеса")
	}

	// Запись действительно удалена.
	if _, err := store.GetManga(42); !errors.Is(err, storage.ErrMangaNotFound) {
		t.Errorf("после DELETE запись осталась: err = %v", err)
	}

	// Повторное удаление — 404.
	rec = doJSON(t, r, http.MethodDelete, "/admin-api/manga/42", access, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("повторный delete: код %d, ожидался 404", rec.Code)
	}
}

// TestCreateMangaValidation — CREATE: обязательные поля и enum.
func TestCreateMangaValidation(t *testing.T) {
	r, _ := testServer(t)
	access, _ := login(t, r)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"полное тело", jsonBody(t, map[string]any{"source_id": 1, "url": "x_1", "title": "Т", "publication_status": "completed"}), http.StatusCreated},
		{"невалидный статус", jsonBody(t, map[string]any{"source_id": 1, "url": "x_2", "title": "Т", "publication_status": "bogus"}), http.StatusBadRequest},
		{"без url", jsonBody(t, map[string]any{"source_id": 1, "title": "Т"}), http.StatusBadRequest},
		{"неизвестный источник", jsonBody(t, map[string]any{"source_id": 99, "url": "x_3", "title": "Т"}), http.StatusBadRequest},
		{"битый JSON", "{oops", http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := doJSON(t, r, http.MethodPost, "/admin-api/manga", access, tc.body)
		if rec.Code != tc.want {
			t.Errorf("%s: код %d, ожидался %d", tc.name, rec.Code, tc.want)
		}
	}
}

// parseJSON — разбор JSON-ответа.
func parseJSON(body string, v any) error {
	return json.Unmarshal([]byte(body), v)
}
