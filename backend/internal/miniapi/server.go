package miniapi

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/miniapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

// Server реализует контракт oapi.ServerInterface из docs/openapi-miniapp.yaml.
var _ oapi.ServerInterface = (*Server)(nil)

const headerInitData = "X-Telegram-Init-Data"

// ctxUserKey — ключ TelegramUser в gin.Context (см. requireInitData).
const ctxUserKey = "miniapi.telegramUser"

type Server struct {
	store Store
	cfg   Config
}

func New(store Store, cfg Config) *http.Server {
	return &http.Server{Addr: cfg.Addr, Handler: newEngine(store, cfg)}
}

func newEngine(store Store, cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{store: store, cfg: cfg}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), s.cors())

	// Обёртка генератора: биндит path/query-параметры по контракту схемы
	// и передаёт их типизированными аргументами в методы Server.
	w := &oapi.ServerInterfaceWrapper{
		Handler:      s,
		ErrorHandler: bindErrorHandler,
	}

	api := r.Group("/mini-api")
	api.Use(s.requireInitData())
	{
		api.GET("/me", w.GetMe)
		api.GET("/manga/search", w.SearchManga)
		api.GET("/manga/:mangaId", w.GetManga)
		api.GET("/subscriptions", w.ListSubscriptions)
		api.PUT("/subscriptions/:mangaId", w.SubscribeManga)
		api.DELETE("/subscriptions/:mangaId", w.UnsubscribeManga)
	}

	return r
}

func bindErrorHandler(c *gin.Context, err error, statusCode int) {
	log.Printf("[miniapi] %s %s: некорректный параметр запроса: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(statusCode, oapi.Error{Error: "неверный формат параметра запроса"})
}

// cors отдаёт заголовки для прод-origin мини-аппа (GitHub Pages). В dev
// AllowedOrigin пуст — фронт ходит через Vite-прокси, это same-origin.
// Preflight отвечает до auth-мидлвари: браузер не отправляет кастомные
// заголовки в OPTIONS.
func (s *Server) cors() gin.HandlerFunc {
	origin := s.cfg.AllowedOrigin
	return func(c *gin.Context) {
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", headerInitData+", Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// requireInitData — аутентификация каждого запроса: проверка подписи и
// свежести initData, затем привязка пользователя к chat_users (создаётся
// или обновляется). Stateless: сессий нет, проверка на каждый запрос.
func (s *Server) requireInitData() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := validateInitData(c.GetHeader(headerInitData), s.cfg.BotToken, s.cfg.InitDataMaxAge)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, oapi.Error{Error: err.Error()})
			return
		}

		// CreateChatUser — upsert по chat_id; как и /start бота, он
		// возвращает пользователя в активное состояние (is_active=TRUE),
		// admin-блокировка (blocked) при этом не трогается.
		if _, err := s.store.CreateChatUser(storage.ChatUser{
			ChatID:   user.ID,
			Username: user.Username,
			IsActive: true,
		}); err != nil {
			log.Printf("[miniapi] привязка chat_users %d: %v", user.ID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, oapi.Error{Error: "внутренняя ошибка"})
			return
		}

		c.Set(ctxUserKey, user)
		c.Next()
	}
}
