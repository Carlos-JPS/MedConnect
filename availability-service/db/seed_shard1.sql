-- Seed demo para availability shard1.
-- Router configurado: partition = CRC32(doctor_id) % 16.
-- Partition map local: particiones 0-7 -> shard0, 8-15 -> shard1.
-- Doctor incluido aquí:
--   4f1cb247-7810-4ae4-9267-a8df9c7a0835 -> partición 12 -> shard1

INSERT INTO doctor_calendars (id, doctor_id, specialty, timezone, appointment_duration_minutes, status)
VALUES
    (
        'da6008c0-f4d0-4210-842e-c64c4287279a',
        '4f1cb247-7810-4ae4-9267-a8df9c7a0835',
        'Medicina interna',
        'America/Santiago',
        30,
        'active'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO availability_slots (id, calendar_id, start_time, end_time, status)
VALUES
    (
        '5cb4ad32-545f-4810-bd7f-0a979e4f25e5',
        'da6008c0-f4d0-4210-842e-c64c4287279a',
        NOW() + INTERVAL '3 days',
        NOW() + INTERVAL '3 days' + INTERVAL '30 minutes',
        'available'
    )
ON CONFLICT (id) DO NOTHING;
