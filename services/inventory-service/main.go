package main

import (
	"context"
	"errors"
	stdlog "log" // Alias standard log to prevent conflict
	"os"
	"os/signal"

	// Local config package

	// OpenTelemetry Kafka instrumentation

	// Official OTel Contrib bridge for Zap

	// OTLP HTTP for Traces

	// SDK for traces
	// Add this for SpanFromContext

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

// NewApplication creates a new Application instance
func NewApplication() *Application {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return &Application{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize sets up all the application components
func (app *Application) Initialize() error {
	// Initialize infrastructure (expensive singletons)
	infra, err := NewInfrastructure(app.ctx)
	if err != nil {
		return err
	}
	app.infra = infra

	// Create service factory
	app.factory = NewServiceFactory(infra)

	app.infra.Logger().Info("Application initialized successfully")
	return nil
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

func main() {
	app := NewApplication()
	defer app.Shutdown()

	if err := app.Initialize(); err != nil {
		stdlog.Fatalf("❌ Failed to initialize application: %v", err)
	}

	if err := app.Run(); err != nil {
		app.infra.Logger().Error("❌ Application error", zap.Error(err))
	}
}
