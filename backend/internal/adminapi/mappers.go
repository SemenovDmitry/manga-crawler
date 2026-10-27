package adminapi

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

var storeErrorStatus = map[error]int{
	storage.ErrSourceNotFound:   http.StatusNotFound,
	storage.ErrMangaNotFound:    http.StatusNotFound,
	storage.ErrChatUserNotFound: http.StatusNotFound,
	storage.ErrMangaDuplicate:   http.StatusConflict,
}

func respondStoreError(c *gin.Context, err error, statusOverride ...int) {
	status := http.StatusInternalServerError
	if len(statusOverride) > 0 {
		status = statusOverride[0]
	} else if known, ok := storeErrorStatus[err]; ok {
		status = known
	} else {
		for sentinel, known := range storeErrorStatus {
			if errors.Is(err, sentinel) {
				status = known
				break
			}
		}
	}

	if status == http.StatusInternalServerError {
		abortInternal(c, err)
		return
	}
	c.JSON(status, oapi.Error{Error: err.Error()})
}

// nullTime — sql.NullTime из storage в указатель схемы (null для невалидных).
func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// deref — значение опционального поля схемы (указателя) в нулевое значение T.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
