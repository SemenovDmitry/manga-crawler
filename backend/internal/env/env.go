// Package env — чтение переменных окружения со значениями по умолчанию.
//
// Незаданная переменная — не ошибка: возвращается значение по умолчанию.
// Некорректное значение тоже не прерывает работу: пишется предупреждение
// в лог и возвращается значение по умолчанию.
package env

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Duration возвращает значение переменной key как time.Duration
// (например "30m", "1h30m") или def, если переменная не задана или некорректна.
func Duration(key string, def time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return def
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("Некорректное значение %s=%q, использую %s", key, value, def)
		return def
	}

	return parsed
}

// String возвращает значение переменной key или def,
// если переменная не задана (пустая).
func String(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return def
}

// Int возвращает значение переменной key как int или def,
// если переменная не задана или некорректна.
func Int(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Некорректное значение %s=%q, использую %d", key, value, def)
		return def
	}

	return parsed
}

// Bool возвращает значение переменной key как bool или def,
// если переменная не задана или некорректна.
func Bool(key string, def bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return def
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Некорректное значение %s=%q, использую %v", key, value, def)
		return def
	}

	return parsed
}
