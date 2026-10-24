package parsers

import "testing"

func TestCleanTitle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<em>Глава с тегом</em>", "Глава с тегом"},
		{"Тест&nbsp;&amp; тест", "Тест\u00a0& тест"}, // &nbsp; → U+00A0, \s его не схлопывает
		{"  несколько   пробелов  ", "несколько пробелов"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := CleanTitle(tc.in); got != tc.want {
			t.Errorf("CleanTitle(%q): получено %q, ожидалось %q", tc.in, got, tc.want)
		}
	}
}
