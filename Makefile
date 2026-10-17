server-start:
	go run ./cmd/bot/main.go

build:
	go build -o manga-crawler-backend ./cmd/bot
