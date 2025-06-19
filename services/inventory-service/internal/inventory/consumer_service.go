package inventory

import (
	"context"
	"errors"

	"inventoryservice/internal/platform/kafka"

	"go.uber.org/zap"
)

type ConsumerService interface {
	Start(ctx context.Context) error
}

type KafkaConsumerService struct {
	consumer       kafka.Consumer
	messageHandler MessageHandler
	logger         *zap.Logger
}

func NewConsumerService(consumer kafka.Consumer, messageHandler MessageHandler, logger *zap.Logger) ConsumerService {
	return &KafkaConsumerService{
		consumer:       consumer,
		messageHandler: messageHandler,
		logger:         logger,
	}
}

func (c *KafkaConsumerService) Start(ctx context.Context) error {
	c.logger.Info("Kafka consumer started. Waiting for messages...")

	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Info("Context done, exiting Kafka read loop.", zap.Error(err))
				break
			}
			c.logger.Error("❌ Error reading from Kafka", zap.Error(err))
			continue
		}

		if err := c.messageHandler.HandleOrderCreated(ctx, *msg); err != nil {
			continue
		}
	}

	c.logger.Info("Consumer service finished. Shutting down...")
	return nil
}
