package miniapi

import (
	"strings"

	"github.com/SemenovDmitry/manga-crawler-backend/internal/miniapi/oapi"
	"github.com/SemenovDmitry/manga-crawler-backend/internal/storage"
)

func toUser(u TelegramUser) oapi.User {
	user := oapi.User{Id: u.ID, FirstName: u.FirstName}
	if u.Username != "" {
		user.Username = &u.Username
	}
	if u.LastName != "" {
		user.LastName = &u.LastName
	}
	if u.LanguageCode != "" {
		user.LanguageCode = &u.LanguageCode
	}
	if u.PhotoURL != "" {
		user.PhotoUrl = &u.PhotoURL
	}
	return user
}

func toManga(m storage.Manga) oapi.Manga {
	return oapi.Manga{
		Id:                m.ID,
		SourceName:        m.SourceName,
		SourceUrl:         fullSourceURL(m),
		Title:             m.Title,
		AltTitles:         m.AltTitles,
		LatestChapter:     m.LatestChapter,
		ChaptersCount:     m.ChaptersCount,
		PublicationStatus: oapi.PublicationStatus(m.PublicationStatus),
	}
}

func toMangaList(items []storage.Manga) oapi.MangaList {
	// Непустой слайс даже для пустого результата — иначе JSON выдаст null.
	list := make([]oapi.Manga, 0, len(items))
	for _, m := range items {
		list = append(list, toManga(m))
	}
	return oapi.MangaList{Manga: list}
}

// fullSourceURL — полная ссылка на мангу у источника.
// Логика telegram.mangaFullURL, продублирована: пакеты не связаны сознательно.
func fullSourceURL(m storage.Manga) string {
	if strings.HasPrefix(m.URL, "http://") || strings.HasPrefix(m.URL, "https://") {
		return m.URL
	}
	return strings.TrimRight(m.BaseURL, "/") + "/" + strings.TrimLeft(m.URL, "/")
}
