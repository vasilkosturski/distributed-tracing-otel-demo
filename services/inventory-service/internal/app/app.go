package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Application represents the main application structure
type Application struct {
	ctx       context.Context
	cancel    context.CancelFunc
	container *Container
}

// NewApplication creates and initializes a new application instance
func NewApplication() (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		ctx:    ctx,
		cancel: cancel,
	}

	container, err := NewContainer(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}
	app.container = container

	app.container.Logger().Info("Application initialized successfully")
	return app, nil
}

// Run starts the main event processing loop
func (app *Application) Run() error {
	return app.container.ConsumerService().Start(app.ctx)
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) {
	app.container.Logger().Info("Application shutdown initiated")

	if app.cancel != nil {
		app.cancel()
	}

	if err := app.container.Close(ctx); err != nil {
		app.container.Logger().Error("Error during container shutdown", zap.Error(err))
	}

	app.container.Logger().Info("Application shutdown complete")
}
