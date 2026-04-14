package grpc

import (
	"log"

	"contracts/orderpb"
	"order-service/internal/stream"
	"order-service/internal/usecase"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderServer struct {
	orderpb.UnimplementedOrderServiceServer
	usecase *usecase.OrderUsecase
	streams *stream.Manager
}

func NewOrderServer(usecase *usecase.OrderUsecase, streams *stream.Manager) *OrderServer {
	return &OrderServer{
		usecase: usecase,
		streams: streams,
	}
}

func (s *OrderServer) SubscribeToOrderUpdates(
	req *orderpb.OrderRequest,
	server gogrpc.ServerStreamingServer[orderpb.OrderStatusUpdate],
) error {
	if req == nil || req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.usecase.GetOrder(req.OrderId)
	if err != nil {
		return status.Error(codes.NotFound, "order not found")
	}

	if err := server.Send(&orderpb.OrderStatusUpdate{
		OrderId: order.ID,
		Status:  order.Status,
	}); err != nil {
		return err
	}

	ch := s.streams.Subscribe(req.OrderId)
	defer s.streams.Unsubscribe(req.OrderId, ch)

	log.Printf("subscriber connected for order %s", req.OrderId)

	for {
		select {
		case <-server.Context().Done():
			log.Printf("subscriber disconnected for order %s", req.OrderId)
			return nil
		case newStatus := <-ch:
			if err := server.Send(&orderpb.OrderStatusUpdate{
				OrderId: req.OrderId,
				Status:  newStatus,
			}); err != nil {
				return err
			}
		}
	}
}

func RegisterOrderGRPCServer(
	grpcServer *gogrpc.Server,
	usecase *usecase.OrderUsecase,
	streams *stream.Manager,
) {
	orderpb.RegisterOrderServiceServer(
		grpcServer,
		NewOrderServer(usecase, streams),
	)
}
