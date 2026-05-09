CREATE TABLE IF NOT EXISTS doctor_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doctor_id UUID NOT NULL,
    specialty VARCHAR(80) NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    appointment_duration_minutes INTEGER NOT NULL DEFAULT 30,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS availability_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID NOT NULL REFERENCES doctor_calendars(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available', -- 'available', 'held', 'booked'
    held_until TIMESTAMPTZ,
    booking_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para mejorar el rendimiento de las búsquedas
CREATE INDEX idx_slots_calendar_id ON availability_slots(calendar_id);
CREATE INDEX idx_slots_start_time ON availability_slots(start_time);
CREATE INDEX idx_calendars_doctor_id ON doctor_calendars(doctor_id);
CREATE INDEX idx_calendars_specialty ON doctor_calendars(specialty);
