FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/notification-service

RUN go mod download
RUN go build -o notification-service ./cmd/notification-service

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/notification-service/notification-service .

CMD ["./notification-service"]