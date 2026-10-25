package adminapi

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/adminapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func (s *Server) Login(c *gin.Context) {
	var req oapi.LoginJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается JSON с полями username и password"})
		return
	}

	if !s.credentialsValid(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, oapi.Error{Error: "неверный логин или пароль"})
		return
	}

	resp, err := s.issueTokens(req.Username)
	if err != nil {
		abortInternal(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) Refresh(c *gin.Context) {
	var req oapi.RefreshJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается JSON с полем refresh_token"})
		return
	}

	oldHash := hashRefreshToken(req.RefreshToken)
	old, err := s.store.GetAdminSession(oldHash)
	if errors.Is(err, storage.ErrSessionNotFound) {
		c.JSON(http.StatusUnauthorized, oapi.Error{Error: "неизвестный refresh-токен"})
		return
	}
	if err != nil {
		abortInternal(c, err)
		return
	}

	now := time.Now()
	if old.Revoked || !now.Before(old.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, oapi.Error{Error: "refresh-токен истёк или отозван"})
		return
	}

	// Одиночный админ: новый access-токен выпускается для того же
	// логина из конфига; refresh-токен с пользователем не связан.
	resp, err := s.issueTokens(s.cfg.Username)
	if err != nil {
		abortInternal(c, err)
		return
	}

	// Атомарно: погасить старый токен и записать новый. Если старый уже
	// погашен конкурентным refresh'ем — отклоняем (повторное использование).
	ok, err := s.store.RotateAdminSession(oldHash, adminSessionFor(resp.RefreshToken, now.Add(s.cfg.RefreshTTL)))
	if err != nil {
		abortInternal(c, err)
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, oapi.Error{Error: "refresh-токен уже использован"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) Logout(c *gin.Context) {
	var req oapi.LogoutJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "ожидается JSON с полем refresh_token"})
		return
	}

	if err := s.store.RevokeAdminSession(hashRefreshToken(req.RefreshToken)); err != nil {
		abortInternal(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) credentialsValid(username, password string) bool {
	usernameOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.cfg.Username)) == 1
	passwordOK := bcrypt.CompareHashAndPassword([]byte(s.cfg.PasswordHash), []byte(password)) == nil
	return usernameOK && passwordOK
}

func (s *Server) issueTokens(username string) (oapi.TokenPair, error) {
	access, _, err := newAccessToken(s.cfg.JWTSecret, username, s.cfg.TokenTTL)
	if err != nil {
		return oapi.TokenPair{}, err
	}

	refresh, refreshHash, err := newRefreshToken()
	if err != nil {
		return oapi.TokenPair{}, err
	}

	if err := s.store.CreateAdminSession(adminSessionFor(refreshHash, time.Now().Add(s.cfg.RefreshTTL))); err != nil {
		return oapi.TokenPair{}, err
	}

	return oapi.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

func adminSessionFor(tokenHash string, expiresAt time.Time) storage.AdminSession {
	return storage.AdminSession{TokenHash: tokenHash, ExpiresAt: expiresAt}
}
