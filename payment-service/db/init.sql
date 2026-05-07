CREATE TABLE payment_methods (
    payment_method_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL,
    type              VARCHAR(50),
    provider          VARCHAR(50),
    last4             VARCHAR(4),
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE payments (
    payment_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id  UUID NOT NULL,
    user_id     UUID NOT NULL,
    amount      DECIMAL(10,2) NOT NULL,
    currency    VARCHAR(10) NOT NULL,
    status      VARCHAR(20) NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transaction_logs (
    transaction_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id         UUID NOT NULL,
    external_reference VARCHAR(100),
    status             VARCHAR(20),
    message            TEXT,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_transaction_payment FOREIGN KEY (payment_id) REFERENCES payments(payment_id)
);

CREATE TABLE refunds (
    refund_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,
    amount     DECIMAL(10,2) NOT NULL,
    reason     TEXT,
    status     VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_refund_payment FOREIGN KEY (payment_id) REFERENCES payments(payment_id)
);