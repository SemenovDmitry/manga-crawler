package grouple

import "testing"

func TestMangaURLPart(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://1.seimanga.me/asura_2024", "asura_2024"},
		{"https://a.zazaza.me/one_piece/", "one_piece"},
		{"https://a.zazaza.me", ""},
	}

	for _, tc := range cases {
		got, err := mangaURLPart(tc.raw)
		if err != nil {
			t.Errorf("%s: unexpected ошибка: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: получено %q, ожидалось %q", tc.raw, got, tc.want)
		}
	}
}

func TestIsMangaSitemapURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://1.seimanga.me/asura_2024", true},
		{"https://1.seimanga.me/one_piece", true},
		{"https://1.seimanga.me/list", false},
		{"https://1.seimanga.me/search", false},
		{"https://1.seimanga.me/cover.jpg", false},
		{"https://1.seimanga.me/catalog/abc", false},
		{"https://1.seimanga.me/", false},
	}

	for _, tc := range cases {
		if got := isMangaSitemapURL(tc.raw); got != tc.want {
			t.Errorf("%s: получено %v, ожидалось %v", tc.raw, got, tc.want)
		}
	}
}

// Статусы атрибутов data-*-status движка Grouple (подтверждено probe:
// "FINISHED" у завершённого «Ганца», "PROGRESS" у идущего «Ван Писа»).
func TestMapEngineStatus(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"FINISHED", "completed"},
		{"PROGRESS", "ongoing"},
		{"PAUSED", "hiatus"},
		{"HIATUS", "hiatus"},
		{"CANCELLED", "cancelled"},
		{"DISCONTINUED", "discontinued"},
		{"DROPPED", "discontinued"},
		{"SOMETHING_NEW", ""},
		{"", ""},
	}

	for _, tc := range cases {
		if got := mapEngineStatus(tc.in); got != tc.want {
			t.Errorf("mapEngineStatus(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
