// example_basic.go: Basic example of iris-provider-slog usage
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/agilira/iris"
	slogprovider "github.com/agilira/iris-provider-slog"
)

func main() {
	println("Starting iris-provider-slog basic example...")

	// Step 1: Create slog provider
	provider := slogprovider.New(1000)
	defer func() {
		if err := provider.Close(); err != nil {
			println("Warning: Failed to close provider:", err.Error())
		}
	}()

	// Step 2: Create Iris logger with provider
	readers := []iris.SyncReader{provider}
	logger, err := iris.NewReaderLogger(iris.Config{
		Output:  iris.WrapWriter(os.Stdout),
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Debug,
	}, readers)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			println("Warning: Failed to close logger:", err.Error())
		}
	}()

	// Step 3: Start the logger
	logger.Start()

	println("✅ Logger created and started")

	// Step 4: Create slog logger using our provider
	slogger := slog.New(provider)

	// Step 5: Use slog normally - but get Iris performance!
	slogger.Debug("Application initializing", "component", "main", "version", "1.0.0")
	slogger.Info("User authentication", "user_id", "12345", "method", "oauth")
	slogger.Warn("Rate limit approaching", "current_rate", 95, "limit", 100)
	slogger.Error("Database connection failed", "error", "timeout", "retry_count", 3)

	// Step 6: Test with different field types
	slogger.Info("Performance metrics",
		slog.String("service", "auth"),
		slog.Int("requests", 1523),
		slog.Duration("avg_response", 45*time.Millisecond),
		slog.Bool("healthy", true),
		slog.Float64("cpu_usage", 23.4))

	// Step 7: Test with groups (slog feature)
	groupLogger := slogger.WithGroup("request").With("request_id", "req-123")
	groupLogger.Info("Processing request", "path", "/api/users", "method", "GET")

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Step 8: Sync to ensure all logs are written
	if err := logger.Sync(); err != nil {
		println("Warning: Failed to sync logger:", err.Error())
	}

	println("✅ All done! Check the JSON output above.")
	println("📊 Your slog logs are now accelerated by Iris's high-performance pipeline!")
}
