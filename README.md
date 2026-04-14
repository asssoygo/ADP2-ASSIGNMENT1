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







# AP2 Assignment 2 — gRPC Microservices (Order & Payment)

## 📌 Overview

This project extends Assignment 1 by introducing **gRPC-based communication** and **real-time streaming** between microservices.

The system follows a **Clean Architecture** approach and consists of two services:

* **Order Service** (REST + gRPC server for streaming)
* **Payment Service** (gRPC server)

---

## 🧱 Architecture

Frontend (REST) → Order Service → gRPC → Payment Service → PostgreSQL

* Order Service handles HTTP requests
* Payment Service processes payments via gRPC
* Internal communication is fully migrated from REST to gRPC

---

## ⚙️ Technologies

* Go (Golang)
* gRPC + Protocol Buffers
* PostgreSQL (Dockerized)
* Gin (HTTP framework)
* Docker Compose

---

## 📦 Services

### 🧾 Order Service

* REST API (Create/Get/Cancel orders)
* gRPC Server (Streaming updates)
* gRPC Client (calls Payment Service)
* Manages order lifecycle

### 💳 Payment Service

* gRPC Server
* Processes payments
* Applies business rules:

  * Amount ≤ 100000 → Authorized
  * Amount > 100000 → Declined

---

## 🔗 gRPC Communication

### Payment RPC

```proto
rpc ProcessPayment(PaymentRequest) returns (PaymentResponse);
```

### Streaming RPC

```proto
rpc SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate);
```

---

## 🔄 Streaming (Real-time Updates)

* Clients subscribe to order updates via gRPC
* When order status changes in DB:

  * Order Service publishes update
  * Subscribers receive updates instantly

✔ Real-time
✔ Based on actual DB changes (not simulated)

---

## 🚀 How to Run

### 1. Start databases

```bash
docker compose up -d
```

### 2. Run Payment Service

```bash
cd payment-service
go run ./cmd/payment-service
```

### 3. Run Order Service

```bash
cd order-service
go run ./cmd/order-service
```

---

## 🧪 Test Scenarios

### ✅ Successful Payment

* amount = 15000
* result: Paid

### ❌ Declined Payment

* amount = 150001
* result: Failed / Declined

---

## 📡 Streaming Test

### 1. Start subscriber

```bash
go run ./cmd/order-subscriber <order_id>
```

### 2. Update order status

```bash
PATCH /orders/:id/status
```

### 3. Result

Subscriber receives updates instantly:

```
status = Paid → Shipped → Delivered
```

---

## 📂 Repository Structure

```
contracts/        # generated protobuf code
order-service/    # order logic + streaming
payment-service/  # payment gRPC server
frontend/         # simple UI
```

---

## 🎯 Key Features

* Contract-first design (proto)
* gRPC server-client architecture
* Clean Architecture preserved
* Real-time streaming
* Database-driven updates

---

## 📊 Status

✔ Assignment 2 complete
✔ gRPC fully implemented
✔ Streaming implemented
✔ Ready for demonstration
