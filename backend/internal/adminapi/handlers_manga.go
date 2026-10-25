package adminapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func (s *Server) ListManga(c *gin.Context, params oapi.ListMangaParams) {
	filter, err := parseMangaFilter(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: err.Error()})
		return
	}

	mangaList, err := s.store.ListManga(filter)
	if err != nil {
		abortInternal(c, err)
		return
	}

	out := make([]oapi.Manga, 0, len(mangaList))
	for _, m := range mangaList {
		out = append(out, toMangaResponse(m))
	}
	c.JSON(http.StatusOK, oapi.MangaList{Manga: out})
}

func parseMangaFilter(p oapi.ListMangaParams) (storage.MangaFilter, error) {
	filter := storage.MangaFilter{Limit: 50} // default из схемы

	if p.SourceId != nil {
		if *p.SourceId <= 0 {
			return storage.MangaFilter{}, errors.New("source_id должен быть положительным числом")
		}
		filter.SourceID = *p.SourceId
	}
	if p.Query != nil && *p.Query != "" {
		filter.Query = *p.Query
	}
	if p.Status != nil {
		if !p.Status.Valid() {
			return storage.MangaFilter{}, errors.New("status должен быть ongoing, completed, hiatus, cancelled или discontinued")
		}
		filter.PublicationStatus = storage.PublicationStatus(*p.Status)
	}
	if p.Limit != nil {
		if *p.Limit <= 0 || *p.Limit > 200 {
			return storage.MangaFilter{}, errors.New("limit должен быть от 1 до 200")
		}
		filter.Limit = *p.Limit
	}
	if p.Offset != nil {
		if *p.Offset < 0 {
			return storage.MangaFilter{}, errors.New("offset должен быть неотрицательным числом")
		}
		filter.Offset = *p.Offset
	}

	return filter, nil
}

func (s *Server) GetManga(c *gin.Context, id oapi.MangaID) {
	m, err := s.store.GetManga(id)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toMangaResponse(m))
}

func (s *Server) CreateManga(c *gin.Context) {
	var req oapi.CreateMangaJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается JSON: source_id, url, title обязательны"})
		return
	}
	if req.SourceId <= 0 || req.Url == "" || req.Title == "" {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "обязательны поля source_id, url и title"})
		return
	}
	if req.PublicationStatus != nil && !req.PublicationStatus.Valid() {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "publication_status должен быть ongoing, completed, hiatus, cancelled или discontinued"})
		return
	}

	// Статус не передан — пустое значение: БД подставит ongoing.
	createStatus := storage.StatusOngoing
	if req.PublicationStatus != nil {
		createStatus = storage.PublicationStatus(*req.PublicationStatus)
	}

	created, err := s.store.CreateManga(storage.Manga{
		SourceID:          req.SourceId,
		URL:               req.Url,
		Title:             req.Title,
		AltTitles:         deref(req.AltTitles),
		LatestChapter:     deref(req.LatestChapter),
		ChaptersCount:     deref(req.ChaptersCount),
		PublicationStatus: createStatus,
	})
	if err != nil {
		respondStoreError(c, err, http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusCreated, toMangaResponse(created))
}

func (s *Server) UpdateManga(c *gin.Context, id oapi.MangaID) {
	var req oapi.UpdateMangaJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается JSON с редактируемыми полями манги"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "поле title обязательно"})
		return
	}
	if req.PublicationStatus != nil && !req.PublicationStatus.Valid() {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "publication_status должен быть ongoing, completed, hiatus, cancelled или discontinued"})
		return
	}

	updated, err := s.store.UpdateManga(id, storage.MangaPatch{
		Title:             &req.Title,
		AltTitles:         req.AltTitles,
		LatestChapter:     req.LatestChapter,
		ChaptersCount:     req.ChaptersCount,
		PublicationStatus: statusToStringPtr(req.PublicationStatus),
	})
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toMangaResponse(updated))
}

func (s *Server) DeleteManga(c *gin.Context, id oapi.MangaID) {
	deleted, err := s.store.DeleteManga(id)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toMangaResponse(deleted))
}

func statusToStringPtr(p *oapi.PublicationStatus) *storage.PublicationStatus {
	if p == nil {
		return nil
	}
	s := storage.PublicationStatus(*p)
	return &s
}

func toMangaResponse(m storage.Manga) oapi.Manga {
	return oapi.Manga{
		Id:                m.ID,
		SourceId:          m.SourceID,
		SourceName:        m.SourceName,
		Url:               m.URL,
		Title:             m.Title,
		AltTitles:         m.AltTitles,
		LatestChapter:     m.LatestChapter,
		ChaptersCount:     m.ChaptersCount,
		PublicationStatus: oapi.PublicationStatus(m.PublicationStatus),
		LastCheckAt:       nullTime(m.LastCheckAt),
		CreatedAt:         nullTime(m.CreatedAt),
		UpdatedAt:         nullTime(m.UpdatedAt),
	}
}
