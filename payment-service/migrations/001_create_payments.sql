CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(100) PRIMARY KEY,
    order_id VARCHAR(100) NOT NULL,
    transaction_id VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(50) NOT NULL
);