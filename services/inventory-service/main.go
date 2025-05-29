package main

import (
	"fmt"
	stdlog "log"
)

func main() {
	if err := run(); err != nil {
		stdlog.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	app := NewApplication()
	defer app.Shutdown()

	if err := app.Initialize(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if err := app.Run(); err != nil {
		return fmt.Errorf("application runtime error: %w", err)
	}

	return nil
}
