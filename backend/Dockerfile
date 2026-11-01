# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o manga-crawler-backend ./cmd/bot
RUN go build -o manga-crawler-crawler ./cmd/crawler
RUN go build -o manga-crawler-adminapi ./cmd/adminapi
RUN go build -o manga-crawler-miniapi ./cmd/miniapi

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/manga-crawler-backend .
COPY --from=builder /app/manga-crawler-crawler .
COPY --from=builder /app/manga-crawler-adminapi .
COPY --from=builder /app/manga-crawler-miniapi .
EXPOSE 8080 8081 8082
ENV TZ=Etc/GMT
CMD ["./manga-crawler-backend"]
