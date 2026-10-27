package parsers

import (
	"fmt"
	"sort"
	"strings"
)

// NormalizeChapters сортирует главы по номеру, если все номера числовые.
func NormalizeChapters(chapters []Chapter) []Chapter {
	if len(chapters) < 2 {
		return chapters
	}

	type chapterWithNumber struct {
		chapter Chapter
		number  float64
	}

	pairs := make([]chapterWithNumber, 0, len(chapters))
	sortable := true
	for _, ch := range chapters {
		value, err := parseChapterNumber(ch.Number)
		if err != nil {
			sortable = false
			break
		}
		pairs = append(pairs, chapterWithNumber{chapter: ch, number: value})
	}

	if !sortable {
		return chapters
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].number < pairs[j].number
	})

	sorted := make([]Chapter, len(chapters))
	for i, p := range pairs {
		sorted[i] = p.chapter
	}

	return sorted
}

// FindNewChapters возвращает главы, которых не было при прошлом проходе
// (knownCount — число глав на прошлой проверке; новые — в конце списка,
// отсортированного по возрастанию номера).
func FindNewChapters(chapters []Chapter, knownCount int) []Chapter {
	newCount := len(chapters) - knownCount
	if newCount <= 0 {
		return nil
	}
	if newCount > len(chapters) {
		newCount = len(chapters)
	}

	return chapters[len(chapters)-newCount:]
}

// FindNewChaptersSince — главы для уведомления подписчикам: список вырос,
// но не всякое увеличение списка — новая глава. Драйвер может впервые
// догрузить старую историю (mgeko: карточка отдаёт 50 последних глав,
// полный список живёт на отдельной странице) — тогда счётчик прыгнет
// на десятки глав, которых подписчикам уведомлять нельзя.
//
// Если последний известный номер (latestNumber) — число, «новыми»
// считаются только главы с номером строго больше него; счётчик в этом
// сценарии не нужен: после догрузки истории он уже не отражает число
// реально увиденных глав. Нечисловой последний номер сравнить не с чем —
// работает старое поведение по счётчику.
func FindNewChaptersSince(chapters []Chapter, latestNumber string, knownCount int) []Chapter {
	latest, err := parseChapterNumber(latestNumber)
	if err != nil {
		return FindNewChapters(chapters, knownCount)
	}

	newChapters := make([]Chapter, 0, len(chapters))
	for _, ch := range chapters {
		number, err := parseChapterNumber(ch.Number)
		if err != nil || number > latest {
			newChapters = append(newChapters, ch)
		}
	}

	return newChapters
}

// parseChapterNumber разбирает номер главы ("12", "14.5") в число.
func parseChapterNumber(number string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(strings.TrimSpace(number), "%g", &value)
	if err != nil {
		return 0, fmt.Errorf("не числовой номер главы: %q", number)
	}
	return value, nil
}
