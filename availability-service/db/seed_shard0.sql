-- Seed demo para availability shard0.
-- Router configurado: partition = CRC32(doctor_id) % 16.
-- Partition map local: particiones 0-7 -> shard0, 8-15 -> shard1.
-- Doctores incluidos aquí:
--   7e0d2ab1-164e-4a28-8b95-f24293dd0e91 -> partición 2 -> shard0
--   d2f50707-24ab-4df8-8a6b-cc12a8c47a91 -> partición 7 -> shard0

INSERT INTO doctor_calendars (id, doctor_id, specialty, timezone, appointment_duration_minutes, status)
VALUES
    (
        '21c1c662-bc12-46c3-89c7-7839f06cc4c1',
        '7e0d2ab1-164e-4a28-8b95-f24293dd0e91',
        'Cardiología',
        'America/Santiago',
        30,
        'active'
    ),
    (
        '50473751-1608-4920-97ef-1ad6af04dac3',
        'd2f50707-24ab-4df8-8a6b-cc12a8c47a91',
        'Traumatología',
        'America/Santiago',
        45,
        'active'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO availability_slots (id, calendar_id, start_time, end_time, status)
VALUES
    (
        '0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21',
        '21c1c662-bc12-46c3-89c7-7839f06cc4c1',
        NOW() + INTERVAL '1 day',
        NOW() + INTERVAL '1 day' + INTERVAL '30 minutes',
        'available'
    ),
    (
        '8d3d26e8-4d55-4d0d-99e8-0ed2133b8c31',
        '50473751-1608-4920-97ef-1ad6af04dac3',
        NOW() + INTERVAL '2 days',
        NOW() + INTERVAL '2 days' + INTERVAL '45 minutes',
        'available'
    )
ON CONFLICT (id) DO NOTHING;
