package crawler

import (
	"context"
	"time"
)

// pacer вводит вежливую паузу между запросами к сайту-источнику
// (Config.RequestDelay), с отменой по контексту. Нулевая задержка
// — пауз нет.
type pacer struct {
	delay time.Duration
}

func (p pacer) wait(ctx context.Context) error {
	if p.delay <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.delay):
		return nil
	}
}
