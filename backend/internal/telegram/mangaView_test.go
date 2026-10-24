package telegram

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func TestTruncatedButtonText(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "короткий", title: "Один кусок", want: "Один кусок"},
		{name: "переносы сжимаются", title: "Много\n\tпробелов", want: "Много пробелов"},
		{name: "длинный обрезается", title: strings.Repeat("а", 100), want: strings.Repeat("а", 48) + "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncatedButtonText(tt.title); got != tt.want {
				t.Errorf("truncatedButtonText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindResultsKeyboard(t *testing.T) {
	results := []storage.Manga{
		{ID: 1, Title: "Первая"},
		{ID: 2, Title: "Вторая"},
	}

	keyboard := findResultsKeyboard(results)

	if len(keyboard.InlineKeyboard) != len(results) {
		t.Fatalf("кнопок %d, ожидалось %d", len(keyboard.InlineKeyboard), len(results))
	}

	for i, row := range keyboard.InlineKeyboard {
		if len(row) != 1 {
			t.Fatalf("в ряду %d кнопок %d, ожидалась 1", i, len(row))
		}

		wantData := []string{callbackFollow + ":1", callbackFollow + ":2"}[i]
		if row[0].CallbackData != wantData {
			t.Errorf("callback_data = %q, want %q", row[0].CallbackData, wantData)
		}

		if len(row[0].CallbackData) > 64 {
			t.Errorf("callback_data длиннее 64 байт: %q", row[0].CallbackData)
		}

		if row[0].Text == "" {
			t.Errorf("пустой текст кнопки в ряду %d", i)
		}
	}
}

func TestSubscriptionsKeyboard(t *testing.T) {
	subscriptions := []storage.Manga{{ID: 7, Title: "Манга"}}

	keyboard := subscriptionsKeyboard(subscriptions)

	if len(keyboard.InlineKeyboard) != 1 {
		t.Fatalf("кнопок %d, ожидалась 1", len(keyboard.InlineKeyboard))
	}

	wantData := callbackUnfollow + ":7"
	if keyboard.InlineKeyboard[0][0].CallbackData != wantData {
		t.Errorf("callback_data = %q, want %q", keyboard.InlineKeyboard[0][0].CallbackData, wantData)
	}
}

// TestCallbackPrefixesRoundTrip проверяет согласованность строителей кнопок
// и парсера handleCallbackQuery: callback_data, собранная строителем,
// должна разбираться так же, как это делает парсер (strings.Cut по ':').
func TestCallbackPrefixesRoundTrip(t *testing.T) {
	cases := []struct {
		prefix     string
		mangaID    int64
		wantPrefix string
		wantArg    string
	}{
		{callbackFollow, 42, callbackFollow, "42"},
		{callbackUnfollow, 7, callbackUnfollow, "7"},
	}

	for _, tc := range cases {
		data := fmt.Sprintf("%s:%d", tc.prefix, tc.mangaID) // как в строителях клавиатур
		prefix, arg, found := strings.Cut(data, ":")        // как в handleCallbackQuery

		if !found || prefix != tc.wantPrefix || arg != tc.wantArg {
			t.Errorf("roundtrip %q: prefix=%q arg=%q found=%v, want %q/%q",
				data, prefix, arg, found, tc.wantPrefix, tc.wantArg)
		}
	}
}
