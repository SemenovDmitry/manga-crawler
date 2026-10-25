package adminapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newAccessToken(secret []byte, username string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	})

	signed, err := tok.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("ошибка подписи access-токена: %w", err)
	}
	return signed, expiresAt, nil
}

func parseAccessToken(secret []byte, tokenString string) (string, error) {
	const op = "ошибка проверки access-токена"

	// WithValidMethods — единственная проверка алгоритма: keyfunc нужна
	// только чтобы вернуть секрет, сам алгоритм ограничен списком выше.
	tok, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	if !tok.Valid {
		return "", fmt.Errorf("%s: токен невалиден", op)
	}

	sub, err := tok.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", fmt.Errorf("%s: пустой subject", op)
	}
	return sub, nil
}

func newRefreshToken() (token string, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("ошибка генерации refresh-токена: %w", err)
	}

	token = hex.EncodeToString(raw)
	tokenHash = hashRefreshToken(token)
	return token, tokenHash, nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
