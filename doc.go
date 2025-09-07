// Package slogprovider provides an iris.SyncReader implementation for Go's standard log/slog package.
//
// This provider enables existing slog-based applications to benefit from Iris's high-performance
// logging pipeline without requiring code changes. It implements both the iris.SyncReader interface
// for integration with Iris and the slog.Handler interface for compatibility with slog.
//
// # Key Features
//
//   - Zero Code Changes: Use existing slog code unchanged with enhanced performance
//   - High Performance: 10-20x faster than standard slog through Iris acceleration
//   - Feature Inheritance: Automatic OpenTelemetry, security, and advanced Iris features
//   - Drop-in Replacement: Simply replace slog.Handler with this provider
//   - Thread Safety: Safe for concurrent access from multiple goroutines
//
// # Performance Characteristics
//
//   - slog Handle: ~60-150 ns/op (compared to ~1000+ ns/op for standard handlers)
//   - Record Conversion: ~500-1000 ns/op with zero additional allocations
//   - Overall: 10-20x faster than standard slog implementations
//
// # Basic Usage
//
//	import (
//	    "log/slog"
//	    "os"
//	    "github.com/agilira/iris"
//	    slogprovider "github.com/agilira/iris-provider-slog"
//	)
//
//	func main() {
//	    // Create provider
//	    provider := slogprovider.New(1000)
//	    defer provider.Close()
//
//	    // Create Iris logger with provider
//	    readers := []iris.SyncReader{provider}
//	    logger, err := iris.NewReaderLogger(iris.Config{
//	        Output:  iris.WrapWriter(os.Stdout),
//	        Encoder: iris.NewJSONEncoder(),
//	        Level:   iris.Info,
//	    }, readers)
//	    if err != nil {
//	        panic(err)
//	    }
//	    defer logger.Close()
//
//	    logger.Start()
//
//	    // Use slog normally - but get Iris performance!
//	    slogger := slog.New(provider)
//	    slogger.Info("User login", "user_id", "12345")
//	}
//
// # Advanced Integration
//
// For advanced features like Loki integration, install the writer module:
//
//	go get github.com/agilira/iris-writer-loki
//
// Then configure multiple outputs:
//
//	import loki "github.com/agilira/iris-writer-loki"
//
//	lokiWriter, err := loki.NewWriter(loki.Config{
//	    Endpoint: "http://loki:3100/loki/api/v1/push",
//	    Labels: map[string]string{"service": "my-app"},
//	})
//	if err != nil {
//	    panic(err)
//	}
//
//	logger, err := iris.NewReaderLogger(iris.Config{
//	    Output: iris.MultiWriter(
//	        iris.WrapWriter(os.Stdout),
//	        lokiWriter,
//	    ),
//	    Encoder: iris.NewJSONEncoder(),
//	}, readers,
//	    iris.WithOTel(),    // OpenTelemetry integration
//	    iris.WithCaller(),  // Caller information
//	)
//
// Now slog gets ALL Iris features automatically:
//   - OpenTelemetry trace correlation
//   - Grafana Loki batching
//   - Automatic secret redaction
//   - High-performance ring buffer
//
// # Architecture
//
// This provider implements the iris.SyncReader interface, allowing slog records
// to be processed by Iris's high-performance pipeline:
//
//	slog.Logger → SlogProvider → iris.SyncReader → Iris Ring Buffer → Features
//
// The provider maintains an internal buffer of slog records and converts them
// to Iris records on demand. This design ensures:
//
//   - Non-blocking slog operations
//   - Efficient batching and processing
//   - Automatic cleanup and resource management
//   - Graceful handling of buffer overflow conditions
//
// # Thread Safety
//
// All provider operations are thread-safe:
//   - Multiple goroutines can call Handle() simultaneously
//   - Read() operations are safe for concurrent access
//   - Close() can be called while other operations are in progress
//   - Internal state is protected with appropriate synchronization
//
// # Buffer Management
//
// The provider uses a buffered channel for record storage:
//   - Buffer size is configurable during construction
//   - Full buffers result in record dropping (non-blocking behavior)
//   - Buffer size should be tuned based on logging volume and processing speed
//   - Recommended buffer sizes: 100-1000 for typical applications, 1000+ for high-volume
//
// # Error Handling
//
// The provider follows Iris patterns for error handling:
//   - Handle() drops records on buffer full rather than blocking
//   - Read() respects context cancellation for graceful shutdown
//   - Close() is idempotent and safe to call multiple times
//   - Conversion errors are handled gracefully with fallback behavior
//
// # Level Mapping
//
// Slog levels are mapped to Iris levels as follows:
//   - slog.LevelDebug → iris.Debug
//   - slog.LevelInfo → iris.Info
//   - slog.LevelWarn → iris.Warn
//   - slog.LevelError → iris.Error
//   - Custom levels are mapped to the nearest Iris level
//
// # Field Conversion
//
// Slog attributes are converted to Iris fields with type preservation:
//   - String values → iris.String
//   - Integer values → iris.Int64
//   - Float values → iris.Float64
//   - Boolean values → iris.Bool
//   - Duration values → iris.Dur
//   - Time values → iris.Time
//   - Other types → iris.String (with String() conversion)
//
// # Dependencies
//
// This package requires:
//   - github.com/agilira/iris (core logging library)
//   - Go's standard log/slog package (Go 1.21+)
//
// No additional dependencies are required for basic functionality.
//
// # License
//
// iris-provider-slog is licensed under the Mozilla Public License 2.0.
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package slogprovider
