package service

import (
	"fmt"

	"github.com/sllanoscaro/payment-service/modules/repository"
)

type PaymentService struct {
	repo      repository.PaymentRepository
	processor PaymentProcessor
}

func NewPaymentService(repo repository.PaymentRepository, processor PaymentProcessor) *PaymentService {
	return &PaymentService{repo: repo, processor: processor}
}

func (s *PaymentService) CreatePayment(bookingID, userID, currency string, amount float64) (*repository.Payment, error) {
	if bookingID == "" || userID == "" {
		return nil, fmt.Errorf("booking_id y user_id son obligatorios")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency es obligatorio")
	}

	p := &repository.Payment{
		BookingID: bookingID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Status:    "PENDING",
	}
	return s.repo.CreatePayment(p)
}

func (s *PaymentService) ProcessPayment(paymentID, paymentMethodID string) (transactionID, status string, err error) {
	payment, err := s.repo.GetPaymentByID(paymentID)
	if err != nil {
		return "", "", fmt.Errorf("pago no encontrado: %w", err)
	}
	if payment.Status != "PENDING" {
		return "", "", fmt.Errorf("el pago no está en estado PENDING (estado actual: %s)", payment.Status)
	}

	result, processorErr := s.processor.Charge(paymentMethodID, payment.Amount, payment.Currency)

	// Determinar estado final independientemente de si el procesador devolvió error
	var finalStatus, message, txID string
	if processorErr != nil {
		finalStatus = "FAILED"
		message = processorErr.Error()
		txID = ""
	} else {
		finalStatus = result.Status
		message = result.Message
		txID = result.TransactionID
	}

	// Registrar el intento en transaction_logs siempre (éxito o fallo)
	logEntry := &repository.TransactionLog{
		PaymentID:         paymentID,
		ExternalReference: txID,
		Status:            finalStatus,
		Message:           message,
	}
	if logErr := s.repo.CreateTransactionLog(logEntry); logErr != nil {
		// Log del error pero no interrumpimos: la actualización de estado es más crítica
		fmt.Printf("advertencia: no se pudo registrar transaction_log: %v\n", logErr)
	}

	// Actualizar estado del pago
	if updateErr := s.repo.UpdatePaymentStatus(paymentID, finalStatus); updateErr != nil {
		return "", "", fmt.Errorf("error actualizando estado del pago: %w", updateErr)
	}

	if processorErr != nil {
		return "", finalStatus, fmt.Errorf("procesador externo falló: %w", processorErr)
	}
	return txID, finalStatus, nil
}

func (s *PaymentService) GetPayment(paymentID string) (*repository.Payment, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("payment_id es obligatorio")
	}
	return s.repo.GetPaymentByID(paymentID)
}

func (s *PaymentService) GetPaymentsByUser(userID string) ([]*repository.Payment, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id es obligatorio")
	}
	return s.repo.GetPaymentsByUserID(userID)
}

func (s *PaymentService) GetPaymentByBooking(bookingID string) (*repository.Payment, error) {
	if bookingID == "" {
		return nil, fmt.Errorf("booking_id es obligatorio")
	}
	return s.repo.GetPaymentByBookingID(bookingID)
}

func (s *PaymentService) RefundPayment(paymentID string, amount float64, reason string) (*repository.Refund, error) {
	payment, err := s.repo.GetPaymentByID(paymentID)
	if err != nil {
		return nil, fmt.Errorf("pago no encontrado: %w", err)
	}
	if payment.Status != "COMPLETED" {
		return nil, fmt.Errorf("sólo se pueden reembolsar pagos en estado COMPLETED (estado actual: %s)", payment.Status)
	}
	if amount <= 0 || amount > payment.Amount {
		return nil, fmt.Errorf("monto de reembolso inválido: debe ser mayor a 0 y no exceder %.2f", payment.Amount)
	}

	refund := &repository.Refund{
		PaymentID: paymentID,
		Amount:    amount,
		Reason:    reason,
		Status:    "REFUNDED",
	}
	created, err := s.repo.CreateRefund(refund)
	if err != nil {
		return nil, err
	}

	// Si el reembolso es total, marcar el pago como reembolsado
	if amount == payment.Amount {
		if err := s.repo.UpdatePaymentStatus(paymentID, "REFUNDED"); err != nil {
			fmt.Printf("advertencia: reembolso creado pero no se pudo actualizar estado del pago: %v\n", err)
		}
	}

	return created, nil
}
