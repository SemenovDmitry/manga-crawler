package storage

import (
	"strings"
	"testing"
)

func TestEscapeLike(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"one piece", "one piece"},
		{"100%", "100\\%"},
		{"under_score", "under\\_score"},
		{`back\slash`, `back\\slash`},
	}

	for _, tc := range cases {
		if got := escapeLike(tc.in); got != tc.want {
			t.Errorf("escapeLike(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}

func TestPublicationStatusValid(t *testing.T) {
	for _, s := range allPublicationStatuses {
		if !s.Valid() {
			t.Errorf("статус из списка допустимых %q помечен невалидным", s)
		}
	}

	for _, s := range []PublicationStatus{"", "Ongoing", "bogus", "завершён"} {
		if s.Valid() {
			t.Errorf("статус %q должен быть невалидным", s)
		}
	}
}

func TestPublicationStatusIsFinal(t *testing.T) {
	cases := []struct {
		status   PublicationStatus
		wantOnce bool
	}{
		{StatusOngoing, false},
		// hiatus — перерыв без даты окончания: публикация может возобновиться.
		{StatusHiatus, false},
		{StatusCompleted, true},
		{StatusDiscontinued, true},
		{"bogus", false},
	}

	for _, tc := range cases {
		if got := tc.status.IsFinal(); got != tc.wantOnce {
			t.Errorf("IsFinal(%q): получено %v, ожидалось %v", tc.status, got, tc.wantOnce)
		}
	}
}

func TestFinalPublicationStatusesSQL(t *testing.T) {
	got := finalPublicationStatusesSQL()
	for _, s := range []PublicationStatus{StatusCompleted, StatusCancelled, StatusDiscontinued} {
		if !strings.Contains(got, string(s)) {
			t.Errorf("в массиве финальных статусов нет %q: %s", s, got)
		}
	}
	// ongoing и hiatus — не финальные: публикация продолжается или может возобновиться.
	for _, s := range []PublicationStatus{StatusOngoing, StatusHiatus} {
		if strings.Contains(got, string(s)) {
			t.Errorf("%q не должен быть в финальных статусах: %s", s, got)
		}
	}
}
