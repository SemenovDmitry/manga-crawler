# syntax=docker/dockerfile:1
FROM golang:1.24.5-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o manga-crawler-backend ./cmd/bot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/manga-crawler-backend .
EXPOSE 8080
ENV TZ=Etc/GMT
CMD ["./manga-crawler-backend"]
