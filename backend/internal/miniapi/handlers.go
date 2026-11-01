package miniapi

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/miniapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

const (
	searchDefaultLimit = 20
	searchMaxLimit     = 50
)

// ctxUser возвращает пользователя, положенного requireInitData.
func ctxUser(c *gin.Context) TelegramUser {
	user, _ := c.MustGet(ctxUserKey).(TelegramUser)
	return user
}

// GetMe — «кто я»: пользователь из проверенного initData.
func (s *Server) GetMe(c *gin.Context) {
	c.JSON(http.StatusOK, toUser(ctxUser(c)))
}

// SearchManga — поиск по названию (аналог /find бота).
func (s *Server) SearchManga(c *gin.Context, params oapi.SearchMangaParams) {
	limit := searchDefaultLimit
	if params.Limit != nil {
		limit = min(max(*params.Limit, 1), searchMaxLimit)
	}

	var results []storage.Manga
	if query := strings.TrimSpace(params.Q); query != "" {
		var err error
		if results, err = s.store.SearchManga(query, limit); err != nil {
			abortInternal(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, toMangaList(results))
}

// GetManga — карточка манги с признаком подписки текущего пользователя.
func (s *Server) GetManga(c *gin.Context, mangaId oapi.MangaID) {
	manga, ok := s.ensureManga(c, mangaId)
	if !ok {
		return
	}

	subscribed, err := s.store.IsSubscribed(ctxUser(c).ID, mangaId)
	if err != nil {
		abortInternal(c, err)
		return
	}

	c.JSON(http.StatusOK, oapi.MangaDetail{Manga: toManga(manga), Subscribed: subscribed})
}

// ListSubscriptions — подписки текущего пользователя (аналог /list бота).
func (s *Server) ListSubscriptions(c *gin.Context) {
	subscriptions, err := s.store.UserSubscriptions(ctxUser(c).ID)
	if err != nil {
		abortInternal(c, err)
		return
	}

	c.JSON(http.StatusOK, toMangaList(subscriptions))
}

// SubscribeManga — подписаться (идемпотентно).
func (s *Server) SubscribeManga(c *gin.Context, mangaId oapi.MangaID) {
	if _, ok := s.ensureManga(c, mangaId); !ok {
		return
	}

	if _, err := s.store.Subscribe(ctxUser(c).ID, mangaId); err != nil {
		abortInternal(c, err)
		return
	}

	c.JSON(http.StatusOK, oapi.SubscriptionState{MangaId: mangaId, Subscribed: true})
}

// UnsubscribeManga — отписаться (идемпотентно).
func (s *Server) UnsubscribeManga(c *gin.Context, mangaId oapi.MangaID) {
	if _, ok := s.ensureManga(c, mangaId); !ok {
		return
	}

	if _, err := s.store.Unsubscribe(ctxUser(c).ID, mangaId); err != nil {
		abortInternal(c, err)
		return
	}

	c.JSON(http.StatusOK, oapi.SubscriptionState{MangaId: mangaId, Subscribed: false})
}

// ensureManga загружает мангу или завершает запрос 404/500.
func (s *Server) ensureManga(c *gin.Context, id int64) (storage.Manga, bool) {
	manga, err := s.store.GetManga(id)
	if errors.Is(err, storage.ErrMangaNotFound) || errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, oapi.Error{Error: "манга не найдена"})
		return storage.Manga{}, false
	}
	if err != nil {
		abortInternal(c, err)
		return storage.Manga{}, false
	}
	return manga, true
}

func abortInternal(c *gin.Context, err error) {
	log.Printf("[miniapi] %v", err)
	c.JSON(http.StatusInternalServerError, oapi.Error{Error: "внутренняя ошибка"})
}
