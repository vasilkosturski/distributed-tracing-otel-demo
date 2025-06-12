package app

import (
	"inventoryservice/internal/application/handlers"
	"inventoryservice/internal/application/services"
	"inventoryservice/internal/domain"
)

// ServiceFactory creates business logic services with their dependencies
type ServiceFactory struct {
	infra *Infrastructure
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(infra *Infrastructure) *ServiceFactory {
	return &ServiceFactory{
		infra: infra,
	}
}

// CreateInventoryService creates a new inventory service instance
func (f *ServiceFactory) CreateInventoryService() domain.InventoryService {
	return services.NewInventoryService(f.infra.Logger(), f.infra.Tracer())
}

// CreateMessageHandler creates a new message handler instance
func (f *ServiceFactory) CreateMessageHandler(inventoryService domain.InventoryService) handlers.MessageHandler {
	return handlers.NewMessageHandler(inventoryService, f.infra.MessageProducer(), f.infra.Logger())
}
