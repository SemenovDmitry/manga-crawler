package parsers

import "testing"

func TestNormalizeChaptersNumeric(t *testing.T) {
	input := []Chapter{
		{Number: "12"},
		{Number: "10"},
		{Number: "14.5"},
		{Number: "11"},
	}

	got := NormalizeChapters(input)
	want := []string{"10", "11", "12", "14.5"}

	if len(got) != len(want) {
		t.Fatalf("количество глав: получено %d, ожидалось %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Number != want[i] {
			t.Errorf("глава %d: получено %q, ожидалось %q", i, got[i].Number, want[i])
		}
	}
}

func TestNormalizeChaptersNonNumericKeepsOrder(t *testing.T) {
	input := []Chapter{
		{Number: "extra"},
		{Number: "5"},
	}

	got := NormalizeChapters(input)
	if len(got) != 2 || got[0].Number != "extra" || got[1].Number != "5" {
		t.Errorf("порядок глав с нечисловым номером должен сохраняться, получено: %+v", got)
	}
}

func TestFindNewChapters(t *testing.T) {
	chapters := []Chapter{
		{Number: "1"},
		{Number: "2"},
		{Number: "3"},
		{Number: "4"},
		{Number: "5"},
	}

	cases := []struct {
		name      string
		known     int
		wantCoins int
		wantFirst string
	}{
		{"нет новых", 5, 0, ""},
		{"две новые", 3, 2, "4"},
		{"все новые", 0, 5, "1"},
		{"счётчик уменьшился", 7, 0, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FindNewChapters(chapters, tc.known)

			if len(got) != tc.wantCoins {
				t.Fatalf("новых глав: получено %d, ожидалось %d", len(got), tc.wantCoins)
			}
			if tc.wantCoins > 0 && got[0].Number != tc.wantFirst {
				t.Errorf("первая новая глава: получено %q, ожидалось %q", got[0].Number, tc.wantFirst)
			}
		})
	}
}
