package inventory

type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

type InventoryReservedEvent struct {
	OrderID string `json:"order_id"`
}
