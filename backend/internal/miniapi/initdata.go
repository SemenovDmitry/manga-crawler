package miniapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Ошибки валидации initData — отдаются клиенту как 401.
var (
	errNoInitData      = errors.New("нет заголовка " + headerInitData)
	errBadInitData     = errors.New("неверный формат initData")
	errBadHash         = errors.New("неверная подпись initData")
	errExpiredInitData = errors.New("истёкший initData — переоткройте мини-апп")
	errNoUser          = errors.New("в initData нет пользователя")
)

// TelegramUser — пользователь из поля user initData.
type TelegramUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username,omitempty"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	PhotoURL     string `json:"photo_url,omitempty"`
}

// validateInitData проверяет подпись и свежесть квитка, извлекает пользователя.
//
// Алгоритм Telegram (docs → Validating data received via the Mini App):
// из всех полей, кроме hash, строится data-check-string («key=<value>»,
// отсортировано по key, разделитель \n); проверяемый hash — hex от
// HMAC-SHA256(data-check-string), ключ — HMAC_SHA256(botToken, "WebAppData").
// Поле signature (если есть) в data-check-string остаётся — исключается
// только hash.
func validateInitData(raw, botToken string, maxAge time.Duration) (TelegramUser, error) {
	if raw == "" {
		return TelegramUser{}, errNoInitData
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return TelegramUser{}, errBadInitData
	}

	hash := values.Get("hash")
	if hash == "" {
		return TelegramUser{}, errBadInitData
	}

	fields := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		fields = append(fields, key+"="+values.Get(key))
	}
	sort.Strings(fields)

	mac := hmac.New(sha256.New, hmacSHA256([]byte(botToken), []byte("WebAppData")))
	mac.Write([]byte(strings.Join(fields, "\n")))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(hash)) {
		return TelegramUser{}, errBadHash
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return TelegramUser{}, errBadInitData
	}
	if time.Since(time.Unix(authDate, 0)) > maxAge {
		return TelegramUser{}, errExpiredInitData
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return TelegramUser{}, errNoUser
	}
	return user, nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
