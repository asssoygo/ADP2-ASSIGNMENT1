package main

import (
	"context"
	"io"
	"log"
	"os"

	"contracts/orderpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/order-subscriber <order_id>")
	}

	orderID := os.Args[1]

	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect: ", err)
	}
	defer conn.Close()

	client := orderpb.NewOrderServiceClient(conn)

	stream, err := client.SubscribeToOrderUpdates(context.Background(), &orderpb.OrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		log.Fatal("failed to subscribe: ", err)
	}

	log.Printf("subscribed to order %s", orderID)

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed")
			return
		}
		if err != nil {
			log.Fatal("stream error: ", err)
		}

		log.Printf("order update: order_id=%s status=%s", update.OrderId, update.Status)
	}
}
