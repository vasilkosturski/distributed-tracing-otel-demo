package main

import (
	"context"
	"errors"
	"log" // Standard log for very early errors
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap" // Import otelzap
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global" // To get the global OTel logger provider
	"go.uber.org/zap"                     // Import zap
	"go.uber.org/zap/zapcore"
)

// ZapLogger will be our global zap logger instance
var ZapLogger *zap.Logger

func main() {
	// Initialize a basic Zap logger for bootstrap errors.
	// This will be replaced by a more sophisticated one in run().
	var initErr error
	ZapLogger, initErr = zap.NewProduction()
	if initErr != nil {
		log.Fatalf("Failed to initialize initial zap logger: %v", initErr) // Use standard log if zap fails
	}
	defer ZapLogger.Sync() // Ensure buffered logs are flushed before exiting.

	if err := run(); err != nil {
		ZapLogger.Fatal("Application failed to run", zap.Error(err))
	}
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, otelSetupErr := setupOTelSDK(ctx)
	if otelSetupErr != nil {
		ZapLogger.Error("Failed to setup OpenTelemetry SDK", zap.Error(otelSetupErr))
		// Continue with console-only ZapLogger, or return error depending on requirements
		// For this demo, we log and continue; otelShutdown will handle partial cleanup.
	} else {
		// OTel SDK setup was successful, create the OTel-bridged Zap logger
		logProvider := global.GetLoggerProvider() // Get the globally set OTel LoggerProvider
		// Use the same instrumentation scope name as previously used in dice.go for otelslog
		instrumentationScopeName := "go.opentelemetry.io/otel/example/dice"
		otelZapCore := otelzap.NewCore(instrumentationScopeName,
			otelzap.WithLoggerProvider(logProvider),
		)

		// For local development, also log to console with a human-readable format or JSON
		consoleEncoderConfig := zap.NewProductionEncoderConfig()
		consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // More readable timestamps
		consoleCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(consoleEncoderConfig), // Or NewConsoleEncoder for plain text
			zapcore.Lock(os.Stdout),
			zap.DebugLevel, // Log all levels to console, or make this configurable
		)

		// Create a Tee core to write to both OTel and console
		teeCore := zapcore.NewTee(otelZapCore, consoleCore)
		ZapLogger = zap.New(teeCore,
			zap.AddCaller(),                                    // Adds file and line number
			zap.AddStacktrace(zapcore.ErrorLevel),              // Adds stacktrace for error logs and above
			zap.Fields(zap.String("service.version", "0.1.0")), // Example of a common field
		)
		ZapLogger.Info("Successfully re-initialized Zap logger with OpenTelemetry bridge and console output.")
	}

	// Handle shutdown properly so nothing leaks.
	// This defer needs to be after ZapLogger is potentially re-initialized
	defer func() {
		// otelShutdown can be called even if otelSetupErr was not nil,
		// as it's designed to shut down what was successfully started.
		shutdownErr := otelShutdown(context.Background())
		if shutdownErr != nil {
			ZapLogger.Error("Error during OpenTelemetry shutdown", zap.Error(shutdownErr))
		}
		// Join original setup error (if any) and shutdown error (if any) with the main function error
		if otelSetupErr != nil { // Ensure otelSetupErr is included if it occurred
			err = errors.Join(err, otelSetupErr)
		}
		if shutdownErr != nil {
			err = errors.Join(err, shutdownErr)
		}
	}()

	// Start HTTP server.
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		ZapLogger.Info("Starting HTTP server", zap.String("address", srv.Addr))
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		if !errors.Is(err, http.ErrServerClosed) { // Don't log ErrServerClosed as a fatal error
			ZapLogger.Error("HTTP server ListenAndServe error", zap.Error(err))
		} else {
			ZapLogger.Info("HTTP server closed gracefully.")
			err = nil // Clear err if it's ErrServerClosed from a graceful shutdown
		}
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
		ZapLogger.Info("Shutdown signal received, initiating graceful shutdown...")
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if shutdownHTTPErr := srv.Shutdown(shutdownCtx); shutdownHTTPErr != nil {
		ZapLogger.Error("HTTP server graceful shutdown failed", zap.Error(shutdownHTTPErr))
		err = errors.Join(err, shutdownHTTPErr)
	} else {
		ZapLogger.Info("HTTP server shutdown complete.")
	}
	return
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	registerHandlers(mux, handleFunc) // Pass mux and handleFunc to a separate function

	handler := otelhttp.NewHandler(mux, "http-server",
		otelhttp.WithMeterProvider(otel.GetMeterProvider()),   // Ensure correct providers are used
		otelhttp.WithTracerProvider(otel.GetTracerProvider()), // if not default global
	)
	return handler
}

// registerHandlers centralizes handler registration
func registerHandlers(mux *http.ServeMux, handleFunc func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request))) {
	handleFunc("/rolldice/", rolldice)
	handleFunc("/rolldice/{player}", rolldice)
	// Add other handlers here if needed
}
