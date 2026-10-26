bot-start:
	go run ./cmd/bot/main.go

build:
	go build -o manga-crawler-backend ./cmd/bot

probe:
	go run ./cmd/probe

crawler-start:
	go run ./cmd/crawler/main.go

schema-dump:
	go run ./cmd/schemadump
	@echo Схема сохранена в docs/schema.sql
