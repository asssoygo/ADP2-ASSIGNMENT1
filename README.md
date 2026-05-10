# AP2 Assignment 4 — Performance Optimization: Redis Caching, Adapter Pattern, Rate Limiter

## Overview

This project extends the event-driven microservices platform (Assignment 3) with performance optimizations:
- **Redis cache-aside** pattern in the Order Service for sub-millisecond reads
- **Redis-backed idempotency** in the Notification Service for durable deduplication across restarts
- **Adapter pattern** for pluggable email providers (Simulated / Real)
- **Exponential backoff retry** for email delivery
- **Redis-based rate limiter** middleware (10 req/min per IP) in the Order Service

## Architecture

```
Client
  |
  v (HTTP + Rate Limiter)
Order Service (:8080)
  |  \
  |   \──── Redis (:6379)  ← cache-aside for GET /orders/:id
  |                          ← rate limiter (INCR + EXPIRE per IP)
  v (gRPC)
Payment Service (:8081 HTTP / :50051 gRPC)
  |
  v (AMQP — queue: payment.completed)
RabbitMQ (:5672 / Management UI :15672)
  |
  v (consumer)
Notification Service (:8082)
  |
  \──── Redis (:6379)  ← idempotency keys (notification:{payment_id}, TTL 24h)
          ↑
     SimulatedEmailSender (EmailSender adapter)
```

### Cache Flow for GET /orders/:id

```
GET /orders/:id
  → Check Redis key "order:{id}"
  → Cache HIT  → return cached order (no DB query)
  → Cache MISS → query PostgreSQL → store in Redis with TTL 5 min → return order

Status change (payment, cancel, manual update):
  → Update PostgreSQL
  → DELETE Redis key "order:{id}"   ← cache invalidation
```

## Services

| Service              | Role                                           | Port(s)               |
|----------------------|------------------------------------------------|-----------------------|
| order-service        | Manages order lifecycle, Redis cache + rate limiter | HTTP :8080, gRPC :50052 |
| payment-service      | Processes payments, publishes events           | HTTP :8081, gRPC :50051 |
| notification-service | Consumes events, retries with backoff, Redis idempotency | HTTP :8082 |
| rabbitmq             | Message broker                                 | AMQP :5672, UI :15672 |
| redis                | Cache + rate limiter + idempotency store       | :6379                 |
| order-postgres       | Order database                                 | :5433                 |
| payment-postgres     | Payment database                               | :5434                 |

## Cache Invalidation Strategy

The Order Service uses **cache-aside** (lazy population) with **write-invalidation**:

1. **Read**: Check Redis first. Cache hit → return immediately. Cache miss → query DB, write to cache, return.
2. **Write**: After any status change (`Paid`, `Failed`, `Cancelled`, or manual update), DELETE the Redis key for that order.
3. **TTL**: All cache entries expire after **5 minutes** as a safety net, even if invalidation is missed.

This ensures reads are fast while mutations always reflect the latest state in the next read.

```go
// Cache key format
cacheKey := fmt.Sprintf("order:%s", id)

// On read
if order, err := u.cache.Get(ctx, cacheKey); err == nil {
    return order  // cache hit
}
order, _ := u.repo.GetByID(id)
u.cache.Set(ctx, cacheKey, order, 5*time.Minute)

// On write
u.cache.Delete(ctx, fmt.Sprintf("order:%s", id))
```

## Retry / Backoff Logic (Notification Service)

The `NotificationUsecase` retries email delivery up to **4 attempts** (1 initial + 3 retries) with **exponential backoff** delays:

| Attempt | Delay before this retry |
|---------|------------------------|
| 1       | none (immediate)       |
| 2       | 2 seconds              |
| 3       | 4 seconds              |
| 4       | 8 seconds              |

If all 4 attempts fail, `ProcessPaymentEvent` returns an error, causing the RabbitMQ consumer to Nack the message (eventually routing it to the Dead Letter Queue after 3 consumer-level retries).

The `SimulatedEmailSender` introduces a **500 ms artificial latency** and a **30% random failure rate** to demonstrate the retry logic in action.

```
[Notification] Attempt 1/4 failed for <id>: simulated email delivery failure
[Notification] Retry 1/3 for <id>, waiting 2s
[Notification] Attempt 2/4 failed for <id>: simulated email delivery failure
[Notification] Retry 2/3 for <id>, waiting 4s
[Notification] Attempt 3/4 succeeded
[Email] Sent to customer@example.com for Order #<id>. Amount: $50000
```

## Idempotency Strategy (Notification Service)

Before sending any email, the service checks a Redis key:

```
key:   "notification:{payment_id}"
value: "1"
TTL:   24 hours
```

**Flow:**
1. `EXISTS notification:{payment_id}` → if key exists, skip and ACK (already processed)
2. If not exists → send email with retry/backoff
3. On success → `SET notification:{payment_id} 1 EX 86400`

This replaces the old in-memory `map[string]bool` with a **durable, distributed** store. Idempotency now survives service restarts and works correctly across multiple replicas.

## Rate Limiter (Order Service)

All Order Service routes are protected by a Redis-based rate limiter using the **INCR + EXPIRE** pattern:

```go
key   := "rate:{clientIP}"
count := redis.INCR(key)
if count == 1 { redis.EXPIRE(key, 1 minute) }
if count > 10 { return 429 Too Many Requests }
```

- **Limit**: 10 requests per minute per IP
- **Window**: sliding 1-minute window (resets after the first request's EXPIRE fires)
- **Response on exceed**: `HTTP 429` with `{"error": "rate limit exceeded"}`

## Adapter Pattern (Email Provider)

The `EmailSender` interface in the usecase layer decouples email delivery from business logic:

```go
type EmailSender interface {
    Send(ctx context.Context, event domain.PaymentEvent) error
}
```

The `PROVIDER_MODE` environment variable selects the implementation:

| `PROVIDER_MODE` | Implementation          | Behaviour                                 |
|-----------------|-------------------------|-------------------------------------------|
| `SIMULATED`     | `SimulatedEmailSender`  | 500 ms sleep + 30% random failure rate    |
| `REAL`          | (extend here)           | Plug in SendGrid, SES, etc.               |

## Dead Letter Queue (DLQ) — unchanged from Assignment 3

```
payment.completed  ──(Nack, no-requeue)──▶  payment.dlx (exchange)
                                                  │
                                                  ▼  (routing key: payment.dead)
                                            payment.dead (queue)
```

## Clean Architecture Structure

```
cmd/                          ← entry point (composition root)
internal/
  domain/                     ← pure entities, no dependencies
  usecase/                    ← business logic + interfaces (OrderRepository,
  │                              CacheRepository, PaymentClient, EmailSender)
  repository/postgres/        ← DB implementation (order/payment services)
  infrastructure/
    cache/                    ← RedisCache (order-service)
    rabbitmq/                 ← publisher / consumer
    email/                    ← SimulatedEmailSender (notification-service)
  transport/                  ← HTTP handlers, gRPC servers, middleware
  app/                        ← dependency injection wiring
```

## How to Run

### Prerequisites

- Docker and Docker Compose

### 1. Start all services

```bash
docker-compose up --build
```

Starts: order-postgres, payment-postgres, redis, rabbitmq, payment-service, order-service, notification-service.

### 2. Run migrations

```bash
docker exec -i payment-postgres psql -U postgres -d payment_db < payment-service/migrations/001_create_payments.sql
docker exec -i order-postgres psql -U postgres -d order_db < order-service/migrations/001_create_orders.sql
```

### 3. Test cache-aside

```bash
# Create an order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-001","item_name":"Laptop","amount":50000}'

# First GET — cache MISS, queries PostgreSQL, stores in Redis
curl http://localhost:8080/orders/<id>

# Second GET — cache HIT, served from Redis
curl http://localhost:8080/orders/<id>
```

### 4. Test rate limiter

```bash
# Run 11 requests quickly — the 11th returns HTTP 429
for i in $(seq 1 11); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/orders/<id>
done
```

### 5. Test retry / backoff

Watch notification-service logs — roughly 30% of emails fail and retry:

```bash
docker logs -f notification-service
```

### 6. RabbitMQ Management UI

Open `http://localhost:15672` — login: `guest` / `guest`

## API

### Order Service (:8080)

| Method | Path                    | Description                   |
|--------|-------------------------|-------------------------------|
| POST   | /orders                 | Create order + payment        |
| GET    | /orders/:id             | Get order (cache-aside)       |
| GET    | /orders?min=&max=       | Get orders by amount range    |
| PATCH  | /orders/:id/cancel      | Cancel a pending order        |
| PATCH  | /orders/:id/status      | Update order status           |

### Notification Service (:8082)

| Method | Path            | Description                       |
|--------|-----------------|-----------------------------------|
| GET    | /notifications  | Recent processed notifications    |

## Business Rules

- `amount > 100000` → payment **Declined**, order **Failed**
- `amount <= 100000` → payment **Authorized**, order **Paid**
- Inter-service gRPC calls have a 2-second timeout
- Cache TTL: **5 minutes** for order entries
- Idempotency TTL: **24 hours** for notification keys
- Rate limit: **10 requests/minute** per IP on all Order Service routes
