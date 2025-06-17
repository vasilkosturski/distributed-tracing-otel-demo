package app

import (
	"context"
	"os"
	"os/signal"
)

// Application holds all the components and manages the application lifecycle
type Application struct {
	ctx       context.Context
	cancel    context.CancelFunc
	container *Container
}

// NewApplication creates and fully initializes a new Application instance
func NewApplication(ctx context.Context) (*Application, error) {
	// Set up signal handling
	appCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	app := &Application{
		ctx:    appCtx,
		cancel: cancel,
	}

	// Initialize container (expensive singletons)
	container, err := NewContainer(app.ctx)
	if err != nil {
		cancel() // Clean up context if initialization fails
		return nil, err
	}
	app.container = container

	app.container.Logger().Info("Application initialized successfully")
	return app, nil
}

// Run starts the main event processing loop
func (app *Application) Run() error {
	return app.container.ConsumerService().Start(app.ctx)
}

// Shutdown gracefully shuts down all application components
func (app *Application) Shutdown() {
	if app.container != nil {
		app.container.Logger().Info("Starting application shutdown...")
	}

	// Cancel context
	if app.cancel != nil {
		app.cancel()
	}

	// Shutdown container
	if app.container != nil {
		app.container.Shutdown(context.Background())
	}
}
