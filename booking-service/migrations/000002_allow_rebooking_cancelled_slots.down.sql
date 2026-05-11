DROP INDEX IF EXISTS idx_appointments_active_slot;

ALTER TABLE appointments
    ADD CONSTRAINT appointments_slot_id_key UNIQUE (slot_id);
