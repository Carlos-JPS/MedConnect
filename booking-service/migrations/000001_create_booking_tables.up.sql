CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    slot_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED')),
    payment_id UUID NULL,
    confirmation_code VARCHAR(30) NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    reserved_until TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_appointments_patient_status
    ON appointments (patient_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_appointments_active_slot
    ON appointments (slot_id)
    WHERE status IN ('PENDING_PAYMENT', 'CONFIRMED');

CREATE TABLE IF NOT EXISTS appointment_events (
    id UUID PRIMARY KEY,
    appointment_id UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('CREATED', 'PAYMENT_APPROVED', 'CONFIRMED', 'CANCELLED', 'EXPIRED')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_appointment_events_appointment_created
    ON appointment_events (appointment_id, created_at ASC);
