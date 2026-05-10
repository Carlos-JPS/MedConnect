ALTER TABLE appointments
    DROP CONSTRAINT IF EXISTS appointments_slot_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_appointments_active_slot
    ON appointments (slot_id)
    WHERE status IN ('PENDING_PAYMENT', 'CONFIRMED');
