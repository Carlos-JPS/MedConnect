package repository

import "time"

type Payment struct {
	PaymentID string
	BookingID string
	UserID    string
	Amount    float64
	Currency  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Refund struct {
	RefundID  string
	PaymentID string
	Amount    float64
	Reason    string
	Status    string
	CreatedAt time.Time
}

type TransactionLog struct {
	TransactionID     string
	PaymentID         string
	ExternalReference string
	Status            string
	Message           string
	CreatedAt         time.Time
}
