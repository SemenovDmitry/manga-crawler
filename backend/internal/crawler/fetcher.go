package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const maxResponseBody = 25 << 20 // 25 МБ

const (
	fetchAttempts       = 3
	fetchRetryBaseDelay = 2 * time.Second
	defaultUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// ErrRedirected — карточка отвечает кросс-доменным редиректом: движок
// перенёс тайтл на зеркало (другой домен, часто и другой слаг), по
// исходному адресу контента больше нет. Такая карточка пропускается.
var ErrRedirected = errors.New("редирект на другой домен")

// Fetcher — HTTP-клиент краулера: скачивание страниц с User-Agent,
// лимитом тела ответа и ретраями перемежающихся сетевых сбоев.
//
// Два клиента: Page — для служебных страниц (sitemap, каталог), где
// редиректы переходятся как обычно; Card — для карточек манги, где
// кросс-доменный редирект не переходит (тайтл переехал на зеркало —
// по исходному адресу контента больше нет), вместо тела возвращается
// ErrRedirected.
type Fetcher struct {
	userAgent string
	page      *http.Client
	card      *http.Client
}

// NewFetcher создаёт Fetcher с обоими клиентами (таймаут 45 секунд).
func NewFetcher() *Fetcher {
	return &Fetcher{
		userAgent: defaultUserAgent,
		page:      &http.Client{Timeout: 45 * time.Second},
		card: &http.Client{
			Timeout: 45 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return http.ErrUseLastResponse // кросс-доменный редирект не переходим
				}
				return nil // редирект в пределах того же домена — обычное дело
			},
		},
	}
}

// Page скачивает служебную страницу (sitemap, страница каталога).
func (f *Fetcher) Page(ctx context.Context, rawURL string) ([]byte, error) {
	return f.fetchWithClient(ctx, f.page, rawURL)
}

// Card скачивает карточку манги; кросс-доменный редирект не
// переходит — вместо тела возвращается ErrRedirected.
// Необязательный uaOverride подменяет стандартный User-Agent
// (источники, отдающие SSR только роботам — см. parsers.CardUserAgent).
func (f *Fetcher) Card(ctx context.Context, rawURL string, uaOverride ...string) ([]byte, error) {
	return f.fetchWithClient(ctx, f.card, rawURL, uaOverride...)
}

// fetchWithClient выполняет GET-запрос указанным клиентом с ретрями:
// перемежающиеся сетевые сбои (EOF, TLS handshake timeout, connection
// reset) повторяются с нарастающей паузой; HTTP-статусы, кросс-доменные
// редиректы и отмена контекста не повторяются — это стабильные ответы,
// повтор не изменит результат.
func (f *Fetcher) fetchWithClient(ctx context.Context, client *http.Client, rawURL string, uaOverride ...string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < fetchAttempts; attempt++ {
		if attempt > 0 {
			delay := fetchRetryBaseDelay << (attempt - 1) // 2с, 4с, 8с...
			log.Printf("Повтор %d/%d через %v: %s", attempt, fetchAttempts-1, delay, rawURL)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		body, err := f.fetchOnce(ctx, client, rawURL, uaOverride...)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !isRetryableFetchErr(err) || ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (f *Fetcher) fetchOnce(ctx context.Context, client *http.Client, rawURL string, uaOverride ...string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	userAgent := f.userAgent
	if len(uaOverride) > 0 && uaOverride[0] != "" {
		userAgent = uaOverride[0]
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	// CheckRedirect вернул ErrUseLastResponse: редирект остался
	// необработанным, отдаём его статус телом не является.
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc, _ := resp.Location()
		return nil, fmt.Errorf("%w: %s -> %s", ErrRedirected, rawURL, loc)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d от %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа %s: %w", rawURL, err)
	}

	return body, nil
}

// isRetryableFetchErr определяет, стоит ли повторять запрос:
// только сетевые сбои соединения и чтения ответа, распознаваемые по
// типам ошибок (а не по тексту); обёртки с контекстом и ошибки DNS
// не повторяются (отмена/таймаут запроса — решение вызывающего,
// DNS-сбой стабилен).
func isRetryableFetchErr(err error) bool {
	if err == nil ||
		errors.Is(err, ErrRedirected) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Сбой разрешения имени (нет такого хост, NXDOMAIN): повтор не поможет.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return false
	}

	// Обрыв потока чтения: соединение закрылось посреди ответа.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Ошибка сетевой операции: connection reset, broken pipe и т.п.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Таймаут сетевой операции (в т.ч. TLS handshake timeout).
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
