FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/order-service

RUN go mod download
RUN go build -o order-service ./cmd/order-service

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order-service/order-service .

CMD ["./order-service"]