INSERT INTO payments (
    payment_id,
    booking_id,
    user_id,
    amount,
    currency,
    status
) VALUES (
    'b7d8f1b0-af07-4e57-8859-e929aa77e2fc',
    '00000000-0000-0000-0000-000000000001',
    '46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235',
    15000,
    'CLP',
    'COMPLETED'
)
ON CONFLICT (payment_id) DO NOTHING;
