INSERT INTO doctor_calendars (
    id,
    doctor_id,
    specialty,
    timezone,
    appointment_duration_minutes,
    status
) VALUES
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
    ),
    (
        'da6008c0-f4d0-4210-842e-c64c4287279a',
        '4f1cb247-7810-4ae4-9267-a8df9c7a0835',
        'Medicina interna',
        'America/Santiago',
        30,
        'active'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO availability_slots (
    id,
    calendar_id,
    start_time,
    end_time,
    status
) VALUES
    (
        '0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21',
        '21c1c662-bc12-46c3-89c7-7839f06cc4c1',
        '2026-05-04T09:00:00-04:00',
        '2026-05-04T09:30:00-04:00',
        'available'
    ),
    (
        '8d3d26e8-4d55-4d0d-99e8-0ed2133b8c31',
        '50473751-1608-4920-97ef-1ad6af04dac3',
        '2026-05-04T10:30:00-04:00',
        '2026-05-04T11:15:00-04:00',
        'available'
    ),
    (
        '5cb4ad32-545f-4810-bd7f-0a979e4f25e5',
        'da6008c0-f4d0-4210-842e-c64c4287279a',
        '2026-05-04T15:45:00-04:00',
        '2026-05-04T16:15:00-04:00',
        'available'
    )
ON CONFLICT (id) DO NOTHING;
