# AP2 Assignment 3 — Event-Driven Architecture with RabbitMQ

## Overview

This project extends the gRPC microservices platform with an event-driven layer using RabbitMQ. When a payment is processed, the Payment Service publishes an event to a queue. The Notification Service consumes that event and logs an email notification.

## Architecture

```
Client
  |
  v (HTTP)
Order Service (:8080)
  |
  v (gRPC)
Payment Service (:8081 HTTP / :50051 gRPC)
  |
  v (AMQP — queue: payment.completed)
RabbitMQ (:5672 / Management UI :15672)
  |
  v (consumer)
Notification Service
```

### Event Flow

```
POST /orders
  → Order created (status: Pending)
  → gRPC call to Payment Service
  → Payment saved to DB (status: Authorized / Declined)
  → Event published to RabbitMQ queue "payment.completed"
  → Notification Service receives event
  → Logs: [Notification] Sent email to {email} for Order #{id}. Amount: ${amount}
  → Manual ACK sent to RabbitMQ
```

## Services

| Service              | Role                                  | Port(s)          |
|----------------------|---------------------------------------|------------------|
| order-service        | Manages order lifecycle               | HTTP :8080, gRPC :50052 |
| payment-service      | Processes payments, publishes events  | HTTP :8081, gRPC :50051 |
| notification-service | Consumes events, logs notifications   | —                |
| rabbitmq             | Message broker                        | AMQP :5672, UI :15672 |
| order-postgres       | Order database                        | :5433            |
| payment-postgres     | Payment database                      | :5434            |

## Event-Driven Design Decisions

### Queue: `payment.completed`

- **Durable queue**: survives RabbitMQ restarts
- **Persistent messages** (`DeliveryMode: Persistent`): messages are written to disk
- **Manual ACK** (consumer side): `msg.Ack(false)` is called only after successful processing; on error, `msg.Nack` requeues the message

### Idempotency

The Notification Service maintains an in-memory `map[string]bool` keyed on `message_id` (which equals the payment's UUID). Before processing, it checks the map:

```go
if u.processed[event.MessageID] {
    log.Printf("[Notification] Duplicate message %s skipped", event.MessageID)
    return nil
}
```

If the same message arrives twice (e.g., due to a network retry), it is silently skipped and ACKed without re-sending the notification.

> Note: This map is in-memory only. On restart, already-processed IDs are forgotten. For production, use a persistent store (Redis, DB).

### ACK Logic

| Scenario                        | Action                          |
|---------------------------------|---------------------------------|
| Message parsed and processed OK | `msg.Ack(false)` — remove from queue |
| JSON unmarshal fails (malformed)| `msg.Nack(false, false)` — discard (no requeue) |
| Processing error                | `msg.Nack(false, true)` — requeue for retry |

ACK is never sent before `ProcessPaymentEvent` returns successfully. This guarantees at-least-once delivery.

### Graceful Shutdown (Notification Service)

```go
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
application.Consumer.Close()  // closes AMQP channel + connection
```

Closing the connection causes the `for msg := range msgs` loop to exit naturally, so the consumer goroutine stops cleanly.

## Dead Letter Queue (DLQ)

### Topology

```
payment.completed  ──(Nack, no-requeue)──▶  payment.dlx (exchange)
                                                  │
                                                  ▼  (routing key: payment.dead)
                                            payment.dead (queue)
```

| RabbitMQ object     | Type     | Purpose                                              |
|---------------------|----------|------------------------------------------------------|
| `payment.dlx`       | Exchange | Direct exchange; receives dead-lettered messages     |
| `payment.dead`      | Queue    | Stores messages that exhausted all retry attempts    |
| `payment.completed` | Queue    | Main queue; configured with `x-dead-letter-exchange` |

### Queue arguments on `payment.completed`

| Argument                      | Value           | Effect                                                     |
|-------------------------------|-----------------|------------------------------------------------------------|
| `x-dead-letter-exchange`      | `payment.dlx`   | Routes rejected messages to the DLX                       |
| `x-dead-letter-routing-key`   | `payment.dead`  | DLX routes to the DLQ by this key                         |
| `x-message-ttl`               | `60000` (1 min) | Safety net: unprocessed messages expire to DLQ after 1 min |

### Retry logic

The consumer tracks per-`message_id` attempt counts in an in-memory map:

```
attempt 1 fails → Nack(requeue=true)  → back to payment.completed
attempt 2 fails → Nack(requeue=true)  → back to payment.completed
attempt 3 fails → Nack(requeue=false) → dead-lettered → payment.dead
```

On the DLQ side a goroutine logs:
```
[DLQ] Message <id> moved to dead letter queue after 3 attempts
```

### Upgrade safety

If `payment.completed` already exists without DLQ args (e.g., from a previous run), RabbitMQ returns a `406 PRECONDITION_FAILED`. The consumer detects this, opens a fresh channel, deletes the stale queue, and re-declares it with the correct args — no manual intervention needed.

### Simulating failure for demo

Any order whose `customer_email` contains `"fail@"` will always return a processing error, triggering the full retry → DLQ path. The email is derived from the order ID in the payment service (`customer-<order_id>@example.com`), so to trigger a DLQ demo you can manually call the payment HTTP endpoint with a `fail@` address — or extend the order service to accept customer emails.

**Quick demo:**
```bash
# Direct payment endpoint — forces a fail@ email
curl -X POST http://localhost:8081/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":"demo-fail","amount":1000,"customer_email":"fail@example.com"}'
```

Then watch notification-service logs:
```
[Notification] Message <id> attempt 1/3 failed, requeuing: simulated processing failure
[Notification] Message <id> attempt 2/3 failed, requeuing: simulated processing failure
[Notification] Message <id> failed 3/3 times — routing to DLQ
[DLQ] Message <id> moved to dead letter queue after 3 attempts
```

## Clean Architecture Structure

Each service follows the same layered structure:

```
cmd/                    ← entry point (composition root)
internal/
  domain/               ← pure entities, no dependencies
  usecase/              ← business logic + interfaces
  repository/postgres/  ← database implementation (payment/order services)
  infrastructure/rabbitmq/ ← RabbitMQ publisher/consumer
  transport/            ← HTTP handlers, gRPC servers
  app/                  ← dependency injection wiring
```

The `EventPublisher` interface is defined in the **usecase** layer. The `rabbitmq.Publisher` struct in the **infrastructure** layer implements it. This keeps the business logic decoupled from the message broker.

## How to Run

### Prerequisites

- Docker and Docker Compose

### 1. Start all services

```bash
docker-compose up --build
```

This starts: order-postgres, payment-postgres, rabbitmq, payment-service, order-service, notification-service.

### 2. Run migrations

```bash
docker exec -i payment-postgres psql -U postgres -d payment_db < payment-service/migrations/001_create_payments.sql
docker exec -i order-postgres psql -U postgres -d order_db < order-service/migrations/001_create_orders.sql
```

### 3. Test the event flow

Create an order (triggers payment → event → notification):

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-001","item_name":"Laptop","amount":50000}'
```

Check notification logs:

```bash
docker logs notification-service
# Expected: [Notification] Sent email to customer-<order_id>@example.com for Order #<order_id>. Amount: $50000
```

### 4. RabbitMQ Management UI

Open `http://localhost:15672` — login: `guest` / `guest`

You can inspect the `payment.completed` queue, message rates, and consumers.

## API

### Order Service (:8080)

| Method | Path                    | Description              |
|--------|-------------------------|--------------------------|
| POST   | /orders                 | Create order + payment   |
| GET    | /orders/:id             | Get order by ID          |
| GET    | /orders?min=&max=       | Get orders by amount range |
| PATCH  | /orders/:id/cancel      | Cancel a pending order   |
| PATCH  | /orders/:id/status      | Update order status      |

### Payment Service (:8081)

| Method | Path                    | Description              |
|--------|-------------------------|--------------------------|
| POST   | /payments               | Process payment directly |
| GET    | /payments/:order_id     | Get payment by order ID  |

## Business Rules

- `amount > 100000` → payment **Declined**, order **Failed**
- `amount <= 100000` → payment **Authorized**, order **Paid**
- Inter-service calls have a 2-second timeout
- Paid/Declined orders cannot be cancelled
