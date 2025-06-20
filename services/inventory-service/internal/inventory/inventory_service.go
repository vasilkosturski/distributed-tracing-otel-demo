package inventory

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type InventoryService struct {
	logger *zap.Logger
	tracer trace.Tracer
}

func NewInventoryService(logger *zap.Logger, tracer trace.Tracer) *InventoryService {
	return &InventoryService{
		logger: logger,
		tracer: tracer,
	}
}

func (s *InventoryService) ProcessOrderCreated(ctx context.Context, order OrderCreatedEvent) (*InventoryReservedEvent, error) {
	_, span := s.tracer.Start(ctx, "inventory_check")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", order.OrderID),
		attribute.String("inventory.operation", "stock_check"),
		attribute.String("service.component", "inventory_manager"),
	)

	s.logger.Info("🔍 Checking inventory for order", zap.String("order_id", order.OrderID))

	time.Sleep(50 * time.Millisecond)

	stockAvailable := true
	reservedQuantity := 1

	span.SetAttributes(
		attribute.Bool("inventory.available", stockAvailable),
		attribute.Int("inventory.reserved_quantity", reservedQuantity),
		attribute.String("inventory.status", "reserved"),
	)

	span.SetStatus(codes.Ok, "Inventory successfully reserved")

	result := &InventoryReservedEvent{
		OrderID: order.OrderID,
	}

	return result, nil
}
