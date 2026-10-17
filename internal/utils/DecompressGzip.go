package utils

import (
	"bytes"
	"compress/gzip"
	"io"
)

// DecompressGzipBody декомпрессирует gzip данные
func DecompressGzipBody(body []byte) ([]byte, error) {
	// Пробуем распаковать как gzip
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		// Если не gzip, возвращаем как есть
		return body, nil
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return body, nil
	}

	return decompressed, nil
}
