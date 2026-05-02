package main

import (
	"context"
	"log"
	"os"

	"contracts/paymentpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/payment-list-client <Authorized|Declined>")
	}

	statusFilter := os.Args[1]

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect: ", err)
	}
	defer conn.Close()

	client := paymentpb.NewPaymentServiceClient(conn)

	resp, err := client.ListPayments(context.Background(), &paymentpb.ListPaymentsRequest{
		Status: statusFilter,
	})
	if err != nil {
		log.Fatal("failed to list payments: ", err)
	}

	log.Printf("payments with status=%s", statusFilter)
	for _, p := range resp.Payments {
		log.Printf("id=%s order_id=%s transaction_id=%s amount=%d status=%s",
			p.Id, p.OrderId, p.TransactionId, p.Amount, p.Status)
	}
}
