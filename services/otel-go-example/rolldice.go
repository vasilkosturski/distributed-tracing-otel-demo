package main

import (
	"fmt"
	"io"

	// "log" // Replaced by main.ZapLogger for errors
	"math/rand"
	"net/http"
	"strconv"

	// For seeding rand if not using Go 1.20+ auto-seeding
	// "go.opentelemetry.io/contrib/bridges/otelslog" // Remove this
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap" // Keep zap if you need it for fields, though logger is main.ZapLogger
)

const instrumentationName = "go.opentelemetry.io/otel/example/dice" // Used for tracer and meter

var (
	tracer = otel.Tracer(instrumentationName)
	meter  = otel.Meter(instrumentationName)
	// logger  = otelslog.NewLogger(name) // Remove this
	rollCnt metric.Int64Counter
)

func init() {
	// Seed random number generator (good practice, though Go 1.20+ auto-seeds global rand)
	// For explicit seeding:
	// rand.Seed(time.Now().UnixNano()) //nolint:staticcheck // For older Go versions or explicit control
	// For Go 1.22+, math/rand/v2 is preferred and auto-seeds.
	// Sticking to math/rand for this example as it was in the original.

	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		// Use main.ZapLogger if available, otherwise panic as it's an init-time critical failure
		if ZapLogger != nil {
			ZapLogger.Panic("Failed to create dice.rolls counter", zap.Error(err))
		} else {
			panic(fmt.Sprintf("Failed to create dice.rolls counter: %v", err))
		}
	}
}

func rolldice(w http.ResponseWriter, r *http.Request) {
	// The otelhttp middleware already starts a span.
	// We get the current span from context to add attributes or create child spans.
	// ctx := r.Context()
	// For a custom child span for the roll logic itself:
	ctx, span := tracer.Start(r.Context(), "rolldice.logic")
	defer span.End()

	// Seed local rand instance for better concurrency if this handler is called often
	// localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	// roll := 1 + localRand.Intn(6)
	roll := 1 + rand.Intn(6) // Using global rand for simplicity as in original

	var msg string
	playerName := r.PathValue("player")
	if playerName != "" {
		msg = fmt.Sprintf("%s is rolling the dice", playerName)
		span.SetAttributes(attribute.String("player.name", playerName))
	} else {
		playerName = "Anonymous" // For consistency in logging/attributes
		msg = "Anonymous player is rolling the dice"
		span.SetAttributes(attribute.String("player.name", playerName))
	}

	// Use the global ZapLogger from the main package
	// The context (ctx) isn't directly passed to ZapLogger.Info, but otelzap handles
	// associating the log with the active span in the context when exporting to OTel.
	ZapLogger.Info(msg,
		zap.Int("result", roll),
		zap.String("player", playerName), // Add player name as a structured field
	)

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

	resp := strconv.Itoa(roll) + "\n"
	if _, err := io.WriteString(w, resp); err != nil {
		// log.Printf("Write failed: %v\n", err) // Old way
		ZapLogger.Error("Failed to write HTTP response",
			zap.Error(err),
			zap.String("player", playerName), // Add context to error log
		)
	}
}
