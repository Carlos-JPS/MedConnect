package handler

import (
	"context"

	"github.com/sllanoscaro/payment-service/modules/repository"
	"github.com/sllanoscaro/payment-service/modules/service"
	pb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	p, err := h.svc.CreatePayment(req.BookingId, req.UserId, req.Currency, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error creando pago: %v", err)
	}
	return &pb.CreatePaymentResponse{
		Payment: toProtoPayment(p),
	}, nil
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	txID, st, err := h.svc.ProcessPayment(req.PaymentId, req.PaymentMethodId)
	if err != nil {
		// El procesador externo puede fallar; devolvemos error claro al caller
		// pero el servicio sigue en pie para otras llamadas
		return nil, status.Errorf(codes.Internal, "error procesando pago: %v", err)
	}
	return &pb.ProcessPaymentResponse{
		PaymentId:     req.PaymentId,
		TransactionId: txID,
		Status:        st,
	}, nil
}

func (h *PaymentHandler) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	p, err := h.svc.GetPayment(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "pago no encontrado: %v", err)
	}
	return &pb.GetPaymentResponse{Payment: toProtoPayment(p)}, nil
}

func (h *PaymentHandler) GetPaymentsByUser(ctx context.Context, req *pb.GetPaymentsByUserRequest) (*pb.GetPaymentsByUserResponse, error) {
	payments, err := h.svc.GetPaymentsByUser(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error obteniendo pagos: %v", err)
	}
	var proto []*pb.Payment
	for _, p := range payments {
		proto = append(proto, toProtoPayment(p))
	}
	return &pb.GetPaymentsByUserResponse{Payments: proto}, nil
}

func (h *PaymentHandler) GetPaymentByBooking(ctx context.Context, req *pb.GetPaymentByBookingRequest) (*pb.GetPaymentByBookingResponse, error) {
	p, err := h.svc.GetPaymentByBooking(req.BookingId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "pago no encontrado para la reserva: %v", err)
	}
	return &pb.GetPaymentByBookingResponse{Payment: toProtoPayment(p)}, nil
}

func (h *PaymentHandler) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
	refund, err := h.svc.RefundPayment(req.PaymentId, req.Amount, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error procesando reembolso: %v", err)
	}
	return &pb.RefundPaymentResponse{
		Refund: &pb.Refund{
			RefundId:  refund.RefundID,
			PaymentId: refund.PaymentID,
			Amount:    refund.Amount,
			Reason:    refund.Reason,
			Status:    refund.Status,
			CreatedAt: refund.CreatedAt.String(),
		},
	}, nil
}

func toProtoPayment(p *repository.Payment) *pb.Payment {
	return &pb.Payment{
		PaymentId: p.PaymentID,
		BookingId: p.BookingID,
		UserId:    p.UserID,
		Amount:    p.Amount,
		Currency:  p.Currency,
		Status:    p.Status,
		CreatedAt: p.CreatedAt.String(),
	}
}
