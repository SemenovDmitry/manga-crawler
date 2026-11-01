package crawler

// Тесты ретрая HTTP-статусов: временные коды (429, 502, 503, 504)
// повторяются с нарастающей паузой, стабильные (403, 404) — нет.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func init() {
	// Паузы ретрая в тестах — миллисекунды, а не реальные секунды.
	fetchRetryBaseDelay = time.Millisecond
}

func TestFetcherStatusRetry(t *testing.T) {
	cases := []struct {
		status   int
		attempts int
	}{
		// Временные — все fetchAttempts попытки.
		{http.StatusTooManyRequests, fetchAttempts},
		{http.StatusBadGateway, fetchAttempts},
		{http.StatusServiceUnavailable, fetchAttempts},
		{http.StatusGatewayTimeout, fetchAttempts},
		// Стабильные — одна попытка, повтор бессмыслен.
		{http.StatusForbidden, 1},
		{http.StatusNotFound, 1},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("HTTP %d — %d попыток", tc.status, tc.attempts), func(t *testing.T) {
			attempts := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			_, err := NewFetcher().Page(context.Background(), srv.URL)

			var statusErr *httpStatusError
			if !errors.As(err, &statusErr) || statusErr.code != tc.status {
				t.Fatalf("ожидалась httpStatusError(%d), получено: %v", tc.status, err)
			}
			if attempts != tc.attempts {
				t.Errorf("попыток: %d, ожидалось %d", attempts, tc.attempts)
			}
		})
	}
}

func TestFetcherRetryRecovers(t *testing.T) {
	// 503, 503, 200: временный сбой не роняет загрузку, если сервер
	// отвечает уже на повторе.
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ок"))
	}))
	defer srv.Close()

	body, err := NewFetcher().Page(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if string(body) != "ок" {
		t.Errorf("тело: %q, ожидалось %q", body, "ок")
	}
	if attempts != 3 {
		t.Errorf("попыток: %d, ожидалось 3 (две 503, затем успех)", attempts)
	}
}
