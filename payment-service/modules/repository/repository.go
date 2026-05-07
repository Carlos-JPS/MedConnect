package repository

type PaymentRepository interface {
	CreatePayment(p *Payment) (*Payment, error)
	GetPaymentByID(paymentID string) (*Payment, error)
	GetPaymentsByUserID(userID string) ([]*Payment, error)
	GetPaymentByBookingID(bookingID string) (*Payment, error)
	UpdatePaymentStatus(paymentID, status string) error
	CreateTransactionLog(log *TransactionLog) error
	CreateRefund(r *Refund) (*Refund, error)
}
