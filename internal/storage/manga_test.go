package storage

import "testing"

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
