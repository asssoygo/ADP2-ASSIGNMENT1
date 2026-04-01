📦 AP2 Assignment 1 — Clean Architecture Microservices (Order & Payment)
📌Overview

This project implements a two-service microservice platform in Go using:

Gin (HTTP framework)
PostgreSQL (Dockerized)
Clean Architecture

The system consists of:

Order Service — manages orders and their lifecycle
Payment Service — processes payments and validates limits

Communication between services is implemented strictly via REST, as required.

🏗 Architecture Decisions

Each service follows Clean Architecture and is divided into layers:

domain/ → entities (Order, Payment)
usecase/ → business logic and rules
repository/ → database interaction
transport/http/ → HTTP handlers (thin layer)
app/ → dependency wiring
cmd/ → entry point (composition root)
Why this architecture?
clear separation of concerns
dependency inversion
testability
maintainability
business logic isolated from frameworks

Handlers are intentionally thin — all logic is in use cases.

🧩 Bounded Contexts

The system is decomposed into two independent contexts.

🟦 Order Context

Responsible for:

creating orders
retrieving orders
cancelling orders
updating order status

Owns:

orders table
order_db
🟩 Payment Context

Responsible for:

payment authorization
transaction creation
enforcing payment limits
retrieving payment status

Owns:

payments table
payment_db
❗ Important Rules

This project avoids:

❌ shared database
❌ shared models
❌ shared packages
❌ direct SQL between services

This prevents the distributed monolith anti-pattern.

🔄 Service Communication Flow
POST /orders
   ↓
Create order (Pending)
   ↓
Call Payment Service (POST /payments)
   ↓
Receive Authorized / Declined
   ↓
Update order → Paid / Failed
⚠️ Failure Scenario

If Payment Service is unavailable:

request does NOT hang
timeout (2 seconds) is triggered
Order Service returns 503 Service Unavailable
order status is set to Failed
Why Failed instead of Pending?
provides explicit feedback to client
avoids uncertain state
simplifies debugging
makes system behavior deterministic
🗄 Database Design

Each service uses its own PostgreSQL database (Docker).

🟦 Order DB
Host: localhost
Port: 5433
DB: order_db
🟩 Payment DB
Host: localhost
Port: 5434
DB: payment_db

👉 This ensures data ownership per service.

📊 Architecture Diagram
Client
  |
  v
Order Service (:8080) ---- REST ----> Payment Service (:8081)
     |                                   |
     v                                   v
  order_db                           payment_db
🚀 How to Run
1. Start PostgreSQL containers
docker run --name order-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123 -e POSTGRES_DB=order_db -p 5433:5432 -d postgres

docker run --name payment-postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123 -e POSTGRES_DB=payment_db -p 5434:5432 -d postgres
2. Run migrations
Order DB
docker exec -i order-postgres psql -U postgres -d order_db < order-service/migrations/001_create_orders.sql
Payment DB
docker exec -i payment-postgres psql -U postgres -d payment_db < payment-service/migrations/001_create_payments.sql
3. Start Payment Service
cd payment-service
go run ./cmd/payment-service
4. Start Order Service
cd order-service
go run ./cmd/order-service
📡 API Examples
Create Order
curl.exe -X POST http://localhost:8080/orders ^
-H "Content-Type: application/json" ^
-d "{\"customer_id\":\"1\",\"item_name\":\"Laptop\",\"amount\":15000}"
Get Order
curl http://localhost:8080/orders/{id}
Cancel Order
curl -X PATCH http://localhost:8080/orders/{id}/cancel
Get Payment
curl http://localhost:8081/payments/{order_id}
Declined Example
curl.exe -X POST http://localhost:8080/orders ^
-H "Content-Type: application/json" ^
-d "{\"customer_id\":\"2\",\"item_name\":\"Phone\",\"amount\":150001}"

Expected result:

Payment → Declined
Order → Failed
💰 Business Rules
money uses int64 (not float)
amount must be > 0
amount > 100000 → Declined
Paid orders cannot be cancelled
timeout ≤ 2 seconds for inter-service calls
🎯 Assignment Compliance

This project satisfies all requirements:

✅ Clean Architecture
✅ thin handlers
✅ business logic in use cases
✅ repository abstraction
✅ real PostgreSQL databases
✅ database per service
✅ no shared code
✅ REST communication
✅ timeout handling
✅ failure scenario (503)
✅ all endpoints implemented
✅ business rules enforced
✅ migrations included
✅ README and diagram
