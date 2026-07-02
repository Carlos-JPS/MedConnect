package domain

import "time"

type Notification struct {
	EventID     string
	BookingID   string
	EventType   string
	RecipientID string
	Channel     string
	Message     string
	Status      string
	Payload     []byte
	CreatedAt   time.Time
}
