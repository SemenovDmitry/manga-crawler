package crawler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPauseGateWithoutPause(t *testing.T) {
	g := &pauseGate{}
	if err := g.yield(context.Background()); err != nil {
		t.Errorf("yield без паузы: получено %v, ожидался мгновенный nil", err)
	}
}

func TestPauseGateBlocksUntilResume(t *testing.T) {
	g := &pauseGate{}
	ack := g.pause()

	returned := make(chan error, 1)
	go func() { returned <- g.yield(context.Background()) }()

	// Подтверждение паузы приходит, как только полный обход дошёл до уступки.
	select {
	case <-ack:
	case <-time.After(time.Second):
		t.Fatal("yield не подтвердил паузу")
	}

	// До resume yield остаётся заблокированным.
	select {
	case err := <-returned:
		t.Fatalf("yield вернулся до resume: %v", err)
	default:
	}

	g.resume()
	select {
	case err := <-returned:
		if err != nil {
			t.Errorf("yield после resume: получено %v, ожидался nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("yield не продолжился после resume")
	}

	// Следующий yield проходит без паузы.
	if err := g.yield(context.Background()); err != nil {
		t.Errorf("yield после снятия паузы: получено %v, ожидался nil", err)
	}
}

func TestPauseGateCanceledContext(t *testing.T) {
	g := &pauseGate{}
	_ = g.pause()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := g.yield(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("yield с отменённым контекстом: получено %v, ожидалась context.Canceled", err)
	}

	// resume после ухода по отмене не должен паниковать.
	g.resume()
}

func TestPauseGateResumeWithoutPause(t *testing.T) {
	g := &pauseGate{}
	g.resume() // идемпотентность: без паузы — нет паники
	if err := g.yield(context.Background()); err != nil {
		t.Errorf("yield после лишнего resume: получено %v, ожидался nil", err)
	}
}

func TestPauseGateNilReceiver(t *testing.T) {
	var g *pauseGate
	if err := g.yield(context.Background()); err != nil {
		t.Errorf("yield по nil-gate: получено %v, ожидался nil", err)
	}
}

func TestFullCrawlTrackerSingleSlot(t *testing.T) {
	var tr fullCrawlTracker

	if !tr.start("A") {
		t.Fatal("первый start должен занять слот")
	}
	busy, name, done := tr.state()
	if !busy || name != "A" || done == nil {
		t.Fatalf("state: busy=%v name=%q done=%v, ожидалось true/A/канал", busy, name, done)
	}
	if tr.start("B") {
		t.Fatal("второй start при занятом слоте должен быть отклонён")
	}

	tr.finish()
	busy, name, done = tr.state()
	if busy || name != "" || done != nil {
		t.Fatalf("state после finish: busy=%v name=%q done=%v, ожидалось false/\"\"/nil", busy, name, done)
	}

	// Слот снова доступен.
	if !tr.start("B") {
		t.Fatal("start после finish должен занимать слот")
	}
	tr.finish()
}

func TestFullCrawlTrackerFinishTwice(t *testing.T) {
	var tr fullCrawlTracker
	if !tr.start("A") {
		t.Fatal("start должен занять слот")
	}
	tr.finish()
	tr.finish() // повторный finish не должен паниковать
}
