package main

import (
	"context"
	"inventoryservice/internal/app"
	stdlog "log"
)

func main() {
	if err := run(); err != nil {
		stdlog.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	application, err := app.NewApplication()
	if err != nil {
		return err
	}
	defer application.Shutdown(ctx)

	return application.Run()
}
