package domain

// OrderCreatedEvent represents an order creation event from the order service
type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

// InventoryReservedEvent represents an inventory reservation event
type InventoryReservedEvent struct {
	OrderID string `json:"order_id"`
}
