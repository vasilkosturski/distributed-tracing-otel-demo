package app

import (
	"inventoryservice/internal/inventory"
)

// ServiceFactory creates business logic services with their dependencies
type ServiceFactory struct {
	container *Container
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(container *Container) *ServiceFactory {
	return &ServiceFactory{
		container: container,
	}
}

// CreateInventoryService creates a new inventory service instance
func (f *ServiceFactory) CreateInventoryService() inventory.Service {
	return inventory.NewService(f.container.Logger(), f.container.Tracer())
}

// CreateMessageHandler creates a new message handler instance
func (f *ServiceFactory) CreateMessageHandler(inventoryService inventory.Service) inventory.MessageHandler {
	return inventory.NewMessageHandler(inventoryService, f.container.MessageProducer(), f.container.Logger())
}
