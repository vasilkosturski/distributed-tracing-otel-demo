package app

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Application holds all the components and manages the application lifecycle
type Application struct {
	ctx     context.Context
	cancel  context.CancelFunc
	infra   *Infrastructure
	factory *ServiceFactory
}

// NewApplication creates and fully initializes a new Application instance
func NewApplication(ctx context.Context) (*Application, error) {
	// Set up signal handling
	appCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	app := &Application{
		ctx:    appCtx,
		cancel: cancel,
	}

	// Initialize infrastructure (expensive singletons)
	infra, err := NewInfrastructure(app.ctx)
	if err != nil {
		cancel() // Clean up context if initialization fails
		return nil, err
	}
	app.infra = infra

	// Create service factory
	app.factory = NewServiceFactory(infra)

	app.infra.Logger().Info("Application initialized successfully")
	return app, nil
}

// Run starts the main event processing loop
func (app *Application) Run() error {
	app.infra.Logger().Info("Kafka consumer started. Waiting for messages...")

	for {
		msg, err := app.infra.MessageConsumer().ReadMessage(app.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				app.infra.Logger().Info("Context done, exiting Kafka read loop.", zap.Error(err))
				break
			}
			app.infra.Logger().Error("❌ Error reading from Kafka", zap.Error(err))
			continue
		}

		// Handle message with request-scoped services
		if err := app.handleMessage(app.ctx, *msg); err != nil {
			// Error is already logged in the handler, just continue to next message
			continue
		}
	}

	app.infra.Logger().Info("Inventory Service event loop finished. Shutting down...")
	return nil
}

// handleMessage processes a single message with request-scoped services
func (app *Application) handleMessage(ctx context.Context, msg kafka.Message) error {
	// Create services for this message processing (request-scoped)
	inventoryService := app.factory.CreateInventoryService()
	messageHandler := app.factory.CreateMessageHandler(inventoryService)

	// Process the message
	return messageHandler.HandleOrderCreated(ctx, msg)
}

// Shutdown gracefully shuts down all application components
func (app *Application) Shutdown() {
	if app.infra != nil {
		app.infra.Logger().Info("Starting application shutdown...")
	}

	// Cancel context
	if app.cancel != nil {
		app.cancel()
	}

	// Shutdown infrastructure
	if app.infra != nil {
		app.infra.Shutdown(context.Background())
	}
}
