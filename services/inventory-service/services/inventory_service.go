package services

import (
	"context"
	"time"

	"inventoryservice/events"
	"inventoryservice/internal/infrastructure/observability"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger interface defines what the service needs for logging
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// Tracer interface defines what the service needs for tracing
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

// InventoryService handles inventory-related business logic
type InventoryService struct {
	logger observability.Logger
	tracer observability.Tracer
}

// NewInventoryService creates a new InventoryService instance with explicit dependencies
func NewInventoryService(logger observability.Logger, tracer observability.Tracer) *InventoryService {
	return &InventoryService{
		logger: logger,
		tracer: tracer,
	}
}

// ProcessOrderCreated processes an OrderCreated event and returns an InventoryReserved event
func (s *InventoryService) ProcessOrderCreated(ctx context.Context, order events.OrderCreatedEvent) (*events.InventoryReservedEvent, error) {
	// Create span for inventory checking
	_, span := s.tracer.Start(ctx, "inventory_check")
	defer span.End()

	// Add custom attributes to the span
	span.SetAttributes(
		attribute.String("order.id", order.OrderID),
		attribute.String("inventory.operation", "stock_check"),
		attribute.String("service.component", "inventory_manager"),
	)

	s.logger.Info("🔍 Checking inventory for order", zap.String("order_id", order.OrderID))

	// Simulate inventory lookup time
	time.Sleep(50 * time.Millisecond)

	// In a real system, you'd query a database or external service here
	// For demo purposes, we'll simulate successful inventory check
	stockAvailable := true
	reservedQuantity := 1

	// Add more business-specific attributes
	span.SetAttributes(
		attribute.Bool("inventory.available", stockAvailable),
		attribute.Int("inventory.reserved_quantity", reservedQuantity),
		attribute.String("inventory.status", "reserved"),
	)

	// Mark the span as successful
	span.SetStatus(codes.Ok, "Inventory successfully reserved")

	// Create the result event
	result := &events.InventoryReservedEvent{
		OrderID: order.OrderID,
	}

	return result, nil
}
