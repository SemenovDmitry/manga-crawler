package crawler

import (
	"context"
	"sync"
)

// pauseGate — согласование «пауза полного обхода»: инкрементальный проход
// просит фоновый полный обход замереть между двумя запросами, проходит сам
// и отпускает полный обход продолжаться с того же места. Запросы к сайтам
// в любой момент идут только из одного места, вежливость не нарушается.
//
// Жизненный цикл: pause() → (полный обход дошёл до yield) → ack →
// инкрементальный проход → resume(). Повторный pause до resume не
// предусмотрен: вызывающий всегда снимает паузу (defer resume).
type pauseGate struct {
	mu       sync.Mutex
	paused   bool
	ackCh    chan struct{} // закрывается полным обходом при входе в паузу
	resumeCh chan struct{} // закрывается при выходе из паузы
}

// pause запрашивает паузу и возвращает канал подтверждения: он закроется,
// когда полный обход дойдёт до ближайшей точки уступки (секунды —
// максимум длительность одного запроса).
func (g *pauseGate) pause() <-chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = true
	g.ackCh = make(chan struct{})
	g.resumeCh = make(chan struct{})
	return g.ackCh
}

// resume снимает паузу (идемпотентно): ждущий yield продолжает работу.
func (g *pauseGate) resume() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = false
	if g.resumeCh != nil {
		close(g.resumeCh)
		g.resumeCh = nil
	}
	g.ackCh = nil
}

// yield — точка уступки полного обхода. Вызывается между запросами к
// сайту: без паузы возвращается мгновенно, с паузой — блокируется до
// resume или отмены контекста. Допустим вызов по nil-указателю (тесты
// собирают lister-ы без gate) — behaves как отсутствие паузы.
func (g *pauseGate) yield(ctx context.Context) error {
	if g == nil {
		return nil
	}

	g.mu.Lock()
	if !g.paused {
		g.mu.Unlock()
		return nil
	}
	// Подтверждаем вход в паузу один раз — первым дошедшим yield-ом.
	if g.ackCh != nil {
		close(g.ackCh)
		g.ackCh = nil
	}
	resume := g.resumeCh
	g.mu.Unlock()

	select {
	case <-resume:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// fullCrawlTracker — не более одного фонового полного обхода одновременно.
// done закрывается при завершении текущего обхода (успешного или нет).
type fullCrawlTracker struct {
	mu    sync.Mutex
	busy  bool
	name  string       // имя источника текущего обхода, для логов
	done  chan struct{} // закрыт в finish()
}

func (t *fullCrawlTracker) start(sourceName string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.busy {
		return false
	}
	t.busy = true
	t.name = sourceName
	t.done = make(chan struct{})
	return true
}

func (t *fullCrawlTracker) finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.busy = false
	t.name = ""
	if t.done != nil {
		close(t.done)
		t.done = nil
	}
}

func (t *fullCrawlTracker) state() (busy bool, name string, done <-chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.busy, t.name, t.done
}
