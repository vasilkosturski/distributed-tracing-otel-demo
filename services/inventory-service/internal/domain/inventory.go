package domain

import (
	"context"
	"inventoryservice/events"
)

// InventoryService defines the core business operations for inventory management.
// This interface belongs to the domain layer and represents pure business logic
// without any infrastructure concerns.
type InventoryService interface {
	ProcessOrderCreated(ctx context.Context, order events.OrderCreatedEvent) (*events.InventoryReservedEvent, error)
}
