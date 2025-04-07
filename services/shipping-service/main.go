package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

const (
	kafkaBrokerAddress = "localhost:9092"
	orderCreatedTopic  = "OrderCreated"
	packagingTopic     = "PackagingCompleted"
	groupID            = "shipping-service-group"
)

type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

type PackagingCompletedEvent struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

func main() {
	fmt.Println("Shipping Service is starting...")

	ctx := context.Background()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBrokerAddress},
		Topic:   orderCreatedTopic,
		GroupID: groupID,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("Error closing Kafka reader: %v", err)
		}
	}()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokerAddress),
		Topic:    packagingTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing Kafka writer: %v", err)
		}
	}()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading from Kafka: %v", err)
			continue
		}

		var order OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			log.Printf("Invalid JSON in OrderCreated event: %v", err)
			continue
		}

		log.Printf("📦 Received OrderCreated: order_id=%s", order.OrderID)

		out := PackagingCompletedEvent{
			OrderID: order.OrderID,
			Status:  "PACKAGED",
		}
		payload, err := json.Marshal(out)
		if err != nil {
			log.Printf("Failed to serialize PackagingCompleted event: %v", err)
			continue
		}

		err = writer.WriteMessages(ctx, kafka.Message{
			Value: payload,
		})
		if err != nil {
			log.Printf("Failed to publish PackagingCompleted: %v", err)
		} else {
			log.Printf("Sent PackagingCompleted: order_id=%s", order.OrderID)
		}
	}
}
