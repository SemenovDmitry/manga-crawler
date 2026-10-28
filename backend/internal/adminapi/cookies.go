package adminapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Auth-сессия живёт в двух HttpOnly-cookie с Path=/admin-api: JS их не видит,
// на другие пути и домены они не уходят. SameSite=Strict закрывает CSRF:
// кросс-сайтовые запросы браузер отправляет без cookie.
const (
	cookieAccess  = "admin_access"
	cookieRefresh = "admin_refresh"
)

// setAuthCookies выпускает пару cookie после login/refresh.
// Max-Age каждой cookie равен TTL соответствующего токена.
func (s *Server) setAuthCookies(c *gin.Context, access, refresh string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(cookieAccess, access, int(s.cfg.TokenTTL.Seconds()), "/admin-api", "", s.cfg.CookieSecure, true)
	c.SetCookie(cookieRefresh, refresh, int(s.cfg.RefreshTTL.Seconds()), "/admin-api", "", s.cfg.CookieSecure, true)
}

// clearAuthCookies гасит обе cookie при logout: Max-Age<1 удаляет cookie
// (http.Cookie с MaxAge=-1 в gin превращается в Max-Age=0 — тоже удаление).
func (s *Server) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(cookieAccess, "", -1, "/admin-api", "", s.cfg.CookieSecure, true)
	c.SetCookie(cookieRefresh, "", -1, "/admin-api", "", s.cfg.CookieSecure, true)
}

// refreshCookie возвращает значение cookie admin_refresh (второе значение —
// была ли cookie передана).
func refreshCookie(c *gin.Context) (string, bool) {
	token, err := c.Cookie(cookieRefresh)
	if err != nil || token == "" {
		return "", false
	}
	return token, true
}
