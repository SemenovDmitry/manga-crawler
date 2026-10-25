package adminapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func (s *Server) ListSources(c *gin.Context, params oapi.ListSourcesParams) {
	filter := storage.SourceFilter{}
	if params.Enabled != nil {
		filter.Enabled = params.Enabled
	}

	sources, err := s.store.GetSources(filter)
	if err != nil {
		abortInternal(c, err)
		return
	}

	out := make([]oapi.Source, 0, len(sources))
	for _, src := range sources {
		out = append(out, toSourceResponse(src))
	}
	c.JSON(http.StatusOK, oapi.SourceList{Sources: out})
}

func (s *Server) SetSourceEnabled(c *gin.Context, id oapi.SourceID) {
	var req oapi.SetSourceEnabledJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается поле enabled (true/false)"})
		return
	}

	src, err := s.store.SetSourceEnabled(id, req.Enabled)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSourceResponse(src))
}

func (s *Server) ResetSourceFullCrawl(c *gin.Context, id oapi.SourceID) {
	src, err := s.store.ResetSourceFullCrawl(id)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSourceResponse(src))
}

func toSourceResponse(src storage.Source) oapi.Source {
	return oapi.Source{
		Id:                    src.ID,
		Name:                  src.Name,
		BaseUrl:               src.BaseURL,
		Engine:                src.Engine,
		Enabled:               src.Enabled,
		FullCrawledAt:         nullTime(src.FullCrawledAt),
		FullCrawlIntervalSecs: src.FullCrawlIntervalSecs,
		NextFullCrawlAt:       nullTime(src.NextFullCrawlAt),
	}
}
