# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o manga-crawler-backend ./cmd/bot
RUN go build -o manga-crawler-crawler ./cmd/crawler

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/manga-crawler-backend .
COPY --from=builder /app/manga-crawler-crawler .
EXPOSE 8080
ENV TZ=Etc/GMT
CMD ["./manga-crawler-backend"]
