package adminapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func (s *Server) ListChatUsers(c *gin.Context, params oapi.ListChatUsersParams) {
	filter := storage.ChatUserFilter{}
	if params.Active != nil {
		filter.Active = params.Active
	}

	users, err := s.store.ListChatUsers(filter)
	if err != nil {
		abortInternal(c, err)
		return
	}

	out := make([]oapi.ChatUser, 0, len(users))
	for _, u := range users {
		out = append(out, toChatUserResponse(u))
	}
	c.JSON(http.StatusOK, oapi.ChatUserList{Users: out})
}

func (s *Server) SetChatUserEnabled(c *gin.Context, chatID oapi.ChatID) {
	var req oapi.SetChatUserEnabledJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается поле enabled (true/false)"})
		return
	}

	user, err := s.store.SetChatUserBlocked(chatID, !req.Enabled)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, toChatUserResponse(user))
}

func toChatUserResponse(u storage.ChatUser) oapi.ChatUser {
	return oapi.ChatUser{
		ChatId:    u.ChatID,
		Username:  u.Username,
		IsActive:  u.IsActive,
		Blocked:   u.Blocked,
		CreatedAt: nullTime(u.CreatedAt),
		UpdatedAt: nullTime(u.UpdatedAt),
	}
}
