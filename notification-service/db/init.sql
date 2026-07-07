CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    booking_id UUID NOT NULL,
    event_type VARCHAR(80) NOT NULL,
    recipient_id UUID NOT NULL,
    channel VARCHAR(40) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(40) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_notifications_booking
    ON notifications (booking_id, created_at);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient
    ON notifications (recipient_id, created_at);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient_unread
    ON notifications (recipient_id, read_at, created_at);
