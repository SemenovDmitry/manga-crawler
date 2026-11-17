package parsers

import "testing"

// Единое для всех движков правило url-части: путь целиком, без
// ограничивающих слэшей. Примеры — с адресов реальных источников.
func TestURLPath(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		// grouple: один сегмент
		{"https://1.seimanga.me/asura_2024", "asura_2024"},
		{"https://a.zazaza.me/one_piece/", "one_piece"},
		{"https://a.zazaza.me", ""},
		// mgeko, flame: два сегмента
		{"https://www.mgeko.cc/manga/j3pu-shadow-slave/", "manga/j3pu-shadow-slave"},
		{"https://flamecomics.xyz/series/2/", "series/2"},
		// mangalib: локаль и id--slug
		{"https://mangalib.me/ru/manga/175317--futari-de-cherry/", "ru/manga/175317--futari-de-cherry"},
		// remanga: подстраница тайтла
		{"https://remanga.org/manga/pmd-white-rose/chapters", "manga/pmd-white-rose/chapters"},
	}

	for _, tc := range cases {
		got, err := URLPath(tc.raw)
		if err != nil {
			t.Errorf("%s: неожиданная ошибка: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: получено %q, ожидалось %q", tc.raw, got, tc.want)
		}
	}
}
