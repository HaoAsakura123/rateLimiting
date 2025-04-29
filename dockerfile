# Этап сборки
FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Сборка статического бинарника
RUN CGO_ENABLED=0 GOOS=linux go build -o ./app/main ./cmd/main.go

# Этап финальный
FROM alpine:latest

RUN apk --no-cache add ca-certificates bash

WORKDIR /app

COPY --from=builder /build/app/main .
COPY .env .

EXPOSE 8080

COPY wait-for-it.sh /wait-for-it.sh
RUN chmod +x /wait-for-it.sh

ENTRYPOINT ["sh", "-c", "/wait-for-it.sh db 5432 && ./main"]