package service

import (
	"fmt"
	"time"
)

type PaymentProcessor interface {
	Charge(paymentMethodID string, amount float64, currency string) (*ChargeResult, error)
}

type ChargeResult struct {
	TransactionID string
	Status        string
	Message       string
}

type Processor struct{}

func NewProcessor() PaymentProcessor {
	return &Processor{}
}

func (p *Processor) Charge(paymentMethodID string, amount float64, currency string) (*ChargeResult, error) {
	if amount <= 0 {
		return &ChargeResult{
			TransactionID: "",
			Status:        "FAILED",
			Message:       "monto inválido rechazado por el procesador",
		}, nil
	}

	return &ChargeResult{
		TransactionID: fmt.Sprintf("TXN-%d", time.Now().UnixNano()),
		Status:        "COMPLETED",
		Message:       "pago procesado exitosamente",
	}, nil
}
