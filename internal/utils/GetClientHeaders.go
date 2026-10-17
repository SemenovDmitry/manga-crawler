package utils

import (
	"net/http"
)

func GetClientHeaders(baseUrl string) http.Header {
	return http.Header{
		"User-Agent":      {GetRandomUserAgent()},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language": {"ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Accept-Encoding": {"gzip, deflate"},
		"Referer":         {baseUrl + "/"},
		"Connection":      {"keep-alive"},
	}
}
