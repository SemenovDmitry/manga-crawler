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

// FindNewChaptersSince — уведомляем только о главах новее последнего
// известного номера: догруженная драйвером старая история (mgeko:
// карточка 50 глав → полный список 129) не должна считаться новыми главами.
func TestFindNewChaptersSince(t *testing.T) {
	chapters := []Chapter{
		{Number: "1"},
		{Number: "36.1"},
		{Number: "50"},
		{Number: "126"},
		{Number: "127"},
	}

	cases := []struct {
		name        string
		latest      string
		knownCount  int
		wantCount   int
		wantNumbers []string
	}{
		{
			name:        "догрузка истории не уведомляет",
			latest:      "127", // в БД уже видели главу 127
			knownCount:  50,    // карточных глав было 50, стало 5 — счётчик сломан
			wantCount:   0,
			wantNumbers: nil,
		},
		{
			name:        "реально новые главы",
			latest:      "125",
			knownCount:  3,
			wantCount:   2,
			wantNumbers: []string{"126", "127"},
		},
		{
			name:        "нечисловой последний номер — счётчик",
			latest:      "особая",
			knownCount:  4,
			wantCount:   1,
			wantNumbers: []string{"127"},
		},
		{
			name:        "пустой последний номер — счётчик",
			latest:      "",
			knownCount:  4,
			wantCount:   1,
			wantNumbers: []string{"127"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FindNewChaptersSince(chapters, tc.latest, tc.knownCount)

			if len(got) != tc.wantCount {
				t.Fatalf("новых глав: получено %d, ожидалось %d", len(got), tc.wantCount)
			}
			for i, want := range tc.wantNumbers {
				if got[i].Number != want {
					t.Errorf("новая глава %d: получено %q, ожидалось %q", i, got[i].Number, want)
				}
			}
		})
	}
}
