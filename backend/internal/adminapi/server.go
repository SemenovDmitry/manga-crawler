package adminapi

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
)

// Server реализует контракт oapi.ServerInterface из docs/openapi.yaml.
var _ oapi.ServerInterface = (*Server)(nil)

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
	r.Use(gin.Logger(), gin.Recovery())

	// Обёртка генератора: биндит path/query-параметры по контракту схемы
	// и передаёт их типизированными аргументами в методы Server.
	w := &oapi.ServerInterfaceWrapper{
		Handler:      s,
		ErrorHandler: bindErrorHandler,
	}

	api := r.Group("/admin-api")
	api.POST("/auth/login", s.Login)
	api.POST("/auth/refresh", s.Refresh)

	auth := api.Group("/")
	auth.Use(s.requireAuth())
	{
		auth.POST("/auth/logout", w.Logout)
		auth.GET("/sources", w.ListSources)
		auth.PATCH("/sources/:id/enabled", w.SetSourceEnabled)
		auth.POST("/sources/:id/recrawl", w.ResetSourceFullCrawl)
		auth.GET("/chat-users", w.ListChatUsers)
		auth.PATCH("/chat-users/:chat_id/enabled", w.SetChatUserEnabled)
		auth.GET("/manga", w.ListManga)
		auth.GET("/manga/:id", w.GetManga)
		auth.POST("/manga", w.CreateManga)
		auth.PATCH("/manga/:id", w.UpdateManga)
		auth.DELETE("/manga/:id", w.DeleteManga)
	}

	return r
}

func bindErrorHandler(c *gin.Context, err error, statusCode int) {
	log.Printf("[adminapi] %s %s: некорректный параметр запроса: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(statusCode, oapi.Error{Error: "неверный формат параметра запроса"})
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Access-токен приходит в HttpOnly-cookie admin_access.
		token, err := c.Cookie(cookieAccess)
		if err != nil || token == "" {
			abortUnauthorized(c, "требуется cookie "+cookieAccess+" — выполните POST /auth/login")
			return
		}

		if _, err := parseAccessToken(s.cfg.JWTSecret, token); err != nil {
			abortUnauthorized(c, "недействительный или истёкший access-токен")
			return
		}

		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	// WWW-Authenticate с cookie-схемой подсказывает клиенту способ авторизации.
	c.Header("WWW-Authenticate", `Cookie realm="admin-api"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, oapi.Error{Error: msg})
}

func abortInternal(c *gin.Context, err error) {
	log.Printf("[adminapi] %v", err)
	c.JSON(http.StatusInternalServerError, oapi.Error{Error: "внутренняя ошибка"})
}
