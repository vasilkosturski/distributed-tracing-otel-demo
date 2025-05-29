package main

import (
	"context"
	"fmt"
	stdlog "log"
)

func main() {
	if err := run(); err != nil {
		stdlog.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	app, err := NewApplication(ctx)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer app.Shutdown()

	if err := app.Run(); err != nil {
		return fmt.Errorf("application runtime error: %w", err)
	}

	return nil
}
