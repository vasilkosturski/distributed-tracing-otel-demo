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
	// Initialize a basic Zap logger for bootstrap errors,
	// or if OTel setup fails and we don't have the OTel core.
	// This will be replaced by a more sophisticated one in run() if OTel setup is successful.
	var initErr error
	ZapLogger, initErr = zap.NewProduction() // Default to a production console logger initially
	if initErr != nil {
		log.Fatalf("Failed to initialize initial zap logger: %v", initErr) // Use standard log if zap fails
	}
	// It's good practice to Sync zap loggers, especially before exit.
	// This defer will catch the Sync for the last assigned ZapLogger.
	defer func() {
		if ZapLogger != nil {
			_ = ZapLogger.Sync()
		}
	}()

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
		// ZapLogger remains the initial zap.NewProduction() logger (console only)
		// We will still attempt to shutdown any partially initialized OTel components.
	} else {
		// OTel SDK setup was successful, create the OTel-bridged Zap logger
		logProvider := global.GetLoggerProvider() // Get the globally set OTel LoggerProvider
		instrumentationScopeName := "go.opentelemetry.io/otel/example/dice"
		otelZapCore := otelzap.NewCore(instrumentationScopeName,
			otelzap.WithLoggerProvider(logProvider),
		)

		// Configure ZapLogger to use ONLY the otelZapCore
		ZapLogger = zap.New(otelZapCore, // Use otelZapCore directly, no Tee, no consoleCore
			zap.AddCaller(),                                    // Adds file and line number
			zap.AddStacktrace(zapcore.ErrorLevel),              // Adds stacktrace for error logs and above
			zap.Fields(zap.String("service.version", "0.1.0")), // Example of a common field
		)
		// This log will now only go to OTel
		ZapLogger.Info("Successfully re-initialized Zap logger with OpenTelemetry bridge ONLY.")
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		if otelShutdown != nil { // Ensure otelShutdown is not nil before calling
			shutdownErr := otelShutdown(context.Background())
			if shutdownErr != nil {
				// Use the current ZapLogger (either console or OTel-bridged)
				ZapLogger.Error("Error during OpenTelemetry shutdown", zap.Error(shutdownErr))
			}
			if shutdownErr != nil { // Make sure to join this error as well
				err = errors.Join(err, shutdownErr)
			}
		}
		// Join original setup error (if any)
		if otelSetupErr != nil {
			err = errors.Join(err, otelSetupErr)
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
	case httpErr := <-srvErr: // Renamed to avoid shadowing outer 'err'
		// Error when starting HTTP server.
		if !errors.Is(httpErr, http.ErrServerClosed) {
			ZapLogger.Error("HTTP server ListenAndServe error", zap.Error(httpErr))
			err = errors.Join(err, httpErr) // Join this error
		} else {
			ZapLogger.Info("HTTP server closed gracefully.")
			// err = nil // Don't set outer err to nil here if it might have other errors
		}
		return // Return to execute defers
	case <-ctx.Done():
		stop()
		ZapLogger.Info("Shutdown signal received, initiating graceful shutdown...")
	}

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

	registerHandlers(mux, handleFunc)

	handler := otelhttp.NewHandler(mux, "http-server",
		otelhttp.WithMeterProvider(otel.GetMeterProvider()),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
	)
	return handler
}

func registerHandlers(mux *http.ServeMux, handleFunc func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request))) {
	handleFunc("/rolldice/", rolldice)
	handleFunc("/rolldice/{player}", rolldice)
}
