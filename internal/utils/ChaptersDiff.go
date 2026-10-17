package utils

import (
	"github.com/SemenovDmitry/manga-crawler-backend/internal/types"
)

func ChaptersDiff(a []types.Chapter, b []types.Chapter) []types.Chapter {
	// Если массив a пустой - нет различий
	if len(a) == 0 {
		return nil
	}

	// Если массив b пустой, то все главы из a - новые
	if len(b) == 0 {
		return a
	}

	// Создаем мапу для быстрого поиска глав в b
	bMap := make(map[string]types.Chapter)
	for _, chapter := range b {
		bMap[chapter.Title] = chapter
	}

	var newChapters []types.Chapter
	var foundMatch bool

	// Идем по главам a сверху вниз
	for _, chapterA := range a {
		// Если еще не нашли совпадение
		if !foundMatch {
			// Проверяем, есть ли эта глава в b
			if chapterB, exists := bMap[chapterA.Title]; exists {
				// Проверяем, совпадает ли URL
				if chapterA.URL == chapterB.URL {
					// Нашли совпадение - останавливаем поиск новых глав
					foundMatch = true
					continue
				} else {
					// Глава с таким же названием, но другим URL - добавляем как новую
					newChapters = append(newChapters, chapterA)
				}
			} else {
				// Глава не найдена в b - добавляем как новую
				newChapters = append(newChapters, chapterA)
			}
		}
	}

	// Если нашли новые главы - возвращаем true и список
	if len(newChapters) > 0 {
		return newChapters
	}

	// Если все главы совпали сверху вниз
	return nil
}
