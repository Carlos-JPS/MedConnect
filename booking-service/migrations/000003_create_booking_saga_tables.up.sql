CREATE TABLE IF NOT EXISTS booking_sagas (
    id UUID PRIMARY KEY,
    booking_id UUID NULL,
    payment_id UUID NULL,
    patient_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    slot_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL CHECK (status IN (
        'STARTED',
        'SLOT_HELD',
        'BOOKING_CREATED',
        'PAYMENT_CREATED',
        'PAYMENT_COMPLETED',
        'SLOT_CONFIRMED',
        'COMPLETED',
        'COMPENSATING',
        'COMPENSATED',
        'FAILED',
        'COMPENSATION_FAILED'
    )),
    current_step VARCHAR(50) NOT NULL DEFAULT '',
    compensation_status VARCHAR(30) NOT NULL DEFAULT 'NOT_REQUIRED' CHECK (compensation_status IN (
        'NOT_REQUIRED',
        'PENDING',
        'IN_PROGRESS',
        'COMPLETED',
        'FAILED'
    )),
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_booking_sagas_booking_id
    ON booking_sagas (booking_id)
    WHERE booking_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_booking_sagas_patient_status
    ON booking_sagas (patient_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_booking_sagas_status_updated
    ON booking_sagas (status, updated_at DESC);

CREATE TABLE IF NOT EXISTS booking_saga_events (
    id UUID PRIMARY KEY,
    saga_id UUID NOT NULL REFERENCES booking_sagas(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    step VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_booking_saga_events_saga_created
    ON booking_saga_events (saga_id, created_at ASC);
