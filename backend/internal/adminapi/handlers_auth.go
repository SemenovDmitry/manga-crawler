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

	access, refresh, err := s.issueTokens(req.Username)
	if err != nil {
		abortInternal(c, err)
		return
	}

	// Токены отдаются только в HttpOnly-cookie — JS их не читает.
	s.setAuthCookies(c, access, refresh)
	c.JSON(http.StatusOK, s.tokenPairResponse())
}

func (s *Server) Refresh(c *gin.Context) {
	// Refresh-токен приходит в cookie admin_refresh, а не в JSON-теле.
	rawRefresh, ok := refreshCookie(c)
	if !ok {
		c.JSON(http.StatusBadRequest, oapi.Error{Error: "отсутствует cookie " + cookieRefresh})
		return
	}

	oldHash := hashRefreshToken(rawRefresh)
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
	access, refresh, err := s.issueTokens(s.cfg.Username)
	if err != nil {
		abortInternal(c, err)
		return
	}

	// Атомарно: погасить старый токен и записать новый. Если старый уже
	// погашен конкурентным refresh'ем — отклоняем (повторное использование).
	rotated, err := s.store.RotateAdminSession(oldHash, adminSessionFor(refresh, now.Add(s.cfg.RefreshTTL)))
	if err != nil {
		abortInternal(c, err)
		return
	}
	if !rotated {
		c.JSON(http.StatusUnauthorized, oapi.Error{Error: "refresh-токен уже использован"})
		return
	}

	s.setAuthCookies(c, access, refresh)
	c.JSON(http.StatusOK, s.tokenPairResponse())
}

func (s *Server) Logout(c *gin.Context) {
	// Cookie может не быть (истекла, стёрта) — logout всё равно идёт
	// по access-cookie через requireAuth и отвечает 204.
	rawRefresh, _ := refreshCookie(c)
	if rawRefresh != "" {
		if err := s.store.RevokeAdminSession(hashRefreshToken(rawRefresh)); err != nil {
			abortInternal(c, err)
			return
		}
	}

	s.clearAuthCookies(c)
	c.Status(http.StatusNoContent)
}

func (s *Server) credentialsValid(username, password string) bool {
	usernameOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.cfg.Username)) == 1
	passwordOK := bcrypt.CompareHashAndPassword([]byte(s.cfg.PasswordHash), []byte(password)) == nil
	return usernameOK && passwordOK
}

// issueTokens выпускает новую пару токенов и записывает refresh-сессию в БД.
// Токены возвращаются строками: их место — HttpOnly-cookie, тело ответа их
// не содержит.
func (s *Server) issueTokens(username string) (access, refresh string, err error) {
	access, _, err = newAccessToken(s.cfg.JWTSecret, username, s.cfg.TokenTTL)
	if err != nil {
		return "", "", err
	}

	refresh, refreshHash, err := newRefreshToken()
	if err != nil {
		return "", "", err
	}

	if err := s.store.CreateAdminSession(adminSessionFor(refreshHash, time.Now().Add(s.cfg.RefreshTTL))); err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

// tokenPairResponse — тело ответа login/refresh: метаданные пары без самих
// токенов (они в cookie).
func (s *Server) tokenPairResponse() oapi.TokenPair {
	return oapi.TokenPair{
		TokenType: "Bearer",
		ExpiresIn: int64(s.cfg.TokenTTL.Seconds()),
	}
}

func adminSessionFor(tokenHash string, expiresAt time.Time) storage.AdminSession {
	return storage.AdminSession{TokenHash: tokenHash, ExpiresAt: expiresAt}
}
