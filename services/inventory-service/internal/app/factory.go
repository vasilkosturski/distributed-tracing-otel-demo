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

func (f *ServiceFactory) CreateConsumerService() inventory.ConsumerService {
	inventoryService := inventory.NewService(f.container.Logger(), f.container.Tracer())
	messageHandler := inventory.NewMessageHandler(inventoryService, f.container.MessageProducer(), f.container.Logger())

	return inventory.NewConsumerService(
		f.container.MessageConsumer(),
		messageHandler,
		f.container.Logger(),
	)
}
