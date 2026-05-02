FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/payment-service

RUN go mod download
RUN go build -o payment-service ./cmd/payment-service

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/payment-service/payment-service .

CMD ["./payment-service"]