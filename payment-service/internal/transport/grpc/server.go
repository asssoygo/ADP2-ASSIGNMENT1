package grpc

import (
	"context"

	"contracts/paymentpb"
	"payment-service/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	usecase *usecase.PaymentUsecase
}

func NewPaymentServer(usecase *usecase.PaymentUsecase) *PaymentServer {
	return &PaymentServer{usecase: usecase}
}

func (s *PaymentServer) ProcessPayment(
	ctx context.Context,
	req *paymentpb.PaymentRequest,
) (*paymentpb.PaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	payment, err := s.usecase.CreatePayment(req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &paymentpb.PaymentResponse{
		Status: payment.Status,
	}, nil
}
