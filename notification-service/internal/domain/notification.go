package domain

import "time"

type Notification struct {
	ID          int64
	EventID     string
	BookingID   string
	EventType   string
	RecipientID string
	Channel     string
	Message     string
	Status      string
	Payload     []byte
	CreatedAt   time.Time
	ReadAt      *time.Time
}
