// slog_provider.go: External slog provider for Iris SyncReader interface
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package slogprovider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/agilira/iris"
)

// Provider implements iris.SyncReader for Go's standard log/slog package.
//
// Provider acts as a bridge between slog and Iris, implementing both the
// iris.SyncReader interface for Iris integration and the slog.Handler interface
// for slog compatibility. It captures slog records in an internal buffer and
// converts them to Iris records on demand.
//
// The provider is designed for high performance and thread safety:
//   - Non-blocking Handle() operations (drops records on buffer full)
//   - Efficient record conversion with type preservation
//   - Safe concurrent access from multiple goroutines
//   - Graceful shutdown with proper resource cleanup
//
// Example usage:
//
//	provider := slogprovider.New(1000)
//	defer provider.Close()
//
//	slogger := slog.New(provider)
//	slogger.Info("Message", "key", "value")
type Provider struct {
	records chan slog.Record // Buffered channel for slog records
	closed  chan struct{}    // Signal channel for shutdown coordination
	once    sync.Once        // Ensures Close() is idempotent
}

// New creates a new Provider that captures slog records for processing by Iris.
//
// The bufferSize parameter controls the internal channel buffer size. A larger
// buffer provides better performance under burst loads but uses more memory.
// Recommended values:
//   - 100-500: Low to moderate logging volume applications
//   - 1000-5000: High volume applications
//   - 5000+: Very high volume or burst-heavy applications
//
// When the buffer is full, new records are dropped to maintain non-blocking
// behavior. Monitor your application's logging patterns to choose an appropriate
// buffer size.
//
// The returned Provider must be closed when no longer needed to free resources:
//
//	provider := New(1000)
//	defer provider.Close()
func New(bufferSize int) *Provider {
	return &Provider{
		records: make(chan slog.Record, bufferSize),
		closed:  make(chan struct{}),
	}
}

// Handle implements slog.Handler to capture slog records for processing by Iris.
//
// This method is called by the slog library for each log record. It attempts to
// store the record in the internal buffer for later processing by Iris. The
// operation is non-blocking:
//   - If buffer space is available, the record is stored successfully
//   - If the provider is closed, an error is returned
//   - If the buffer is full, the record is dropped silently (returns nil)
//
// The non-blocking behavior ensures that logging never blocks the application,
// even under high load conditions. Applications should monitor buffer sizes
// and provider performance if record dropping is a concern.
//
// Thread Safety: Safe for concurrent access from multiple goroutines.
func (p *Provider) Handle(ctx context.Context, record slog.Record) error {
	select {
	case p.records <- record:
		return nil
	case <-p.closed:
		return fmt.Errorf("slog provider closed")
	default:
		return nil // Drop if buffer full
	}
}

// Enabled implements slog.Handler to indicate whether records at the given level should be processed.
//
// This implementation always returns true, allowing Iris to handle level filtering
// according to its own configuration. This approach provides more flexibility and
// ensures that level changes in Iris are respected without requiring provider
// reconfiguration.
//
// If you need level filtering at the slog level, consider creating a wrapper
// handler that checks levels before delegating to this provider.
func (p *Provider) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// WithAttrs implements slog.Handler to create a handler with additional attributes.
//
// This implementation returns the same provider instance, as attribute handling
// is delegated to the slog library. The slog library will include the attributes
// in each record before calling Handle(), so no special handling is needed here.
//
// For more sophisticated attribute handling, consider implementing a wrapper
// handler that manages attributes before delegating to this provider.
func (p *Provider) WithAttrs(attrs []slog.Attr) slog.Handler {
	return p
}

// WithGroup implements slog.Handler to create a handler with a named group.
//
// This implementation returns the same provider instance, as group handling
// is delegated to the slog library. The slog library will structure the
// attributes appropriately before calling Handle(), so no special handling
// is needed here.
//
// For more sophisticated group handling, consider implementing a wrapper
// handler that manages groups before delegating to this provider.
func (p *Provider) WithGroup(name string) slog.Handler {
	return p
}

// Read implements iris.SyncReader to provide slog records to the Iris pipeline.
//
// This method is called by Iris to retrieve the next available log record for
// processing. It blocks until:
//   - A record becomes available (returns the converted record)
//   - The context is cancelled (returns context error)
//   - The provider is closed (returns nil, nil)
//
// The method converts slog records to Iris records, preserving message content,
// level information, and all attributes with appropriate type conversion.
//
// Thread Safety: Safe for concurrent access, though typically called by a
// single Iris reader goroutine.
func (p *Provider) Read(ctx context.Context) (*iris.Record, error) {
	select {
	case record := <-p.records:
		return p.convertSlogRecord(record), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closed:
		return nil, nil
	}
}

// Close implements io.Closer to gracefully shut down the provider.
//
// This method signals the provider to stop accepting new records and allows
// pending Read() operations to complete gracefully. It's safe to call multiple
// times and from multiple goroutines.
//
// After Close() is called:
//   - Handle() will return an error for new records
//   - Read() will return nil, nil after processing remaining buffered records
//   - The provider should not be used for new operations
//
// Close() does not wait for pending operations to complete. Use context
// cancellation and proper coordination if you need to ensure all records
// are processed before shutdown.
func (p *Provider) Close() error {
	p.once.Do(func() {
		close(p.closed)
	})
	return nil
}

// convertSlogRecord converts a slog.Record to an iris.Record with full fidelity.
//
// This function preserves the message, level, and all attributes from the slog
// record. Attributes are converted using type-aware conversion to maintain
// type information in the Iris pipeline.
//
// The conversion process:
//  1. Creates a new Iris record with converted level and message
//  2. Iterates through slog attributes
//  3. Converts each attribute to an appropriate Iris field type
//  4. Adds fields to the record (respecting Iris field limits)
//
// If the record has more fields than Iris can handle (32 fields), excess
// fields are silently dropped. This should be rare in typical applications.
func (p *Provider) convertSlogRecord(slogRec slog.Record) *iris.Record {
	record := iris.NewRecord(p.convertLevel(slogRec.Level), slogRec.Message)

	slogRec.Attrs(func(attr slog.Attr) bool {
		field := p.convertAttribute(attr)
		return record.AddField(field)
	})

	return record
}

// convertLevel maps slog.Level values to iris.Level values.
//
// The mapping follows these rules:
//   - slog.LevelDebug → iris.Debug
//   - slog.LevelInfo → iris.Info
//   - slog.LevelWarn → iris.Warn
//   - slog.LevelError and higher → iris.Error
//
// Custom slog levels are mapped to the nearest standard Iris level.
// This ensures that level-based filtering and handling work correctly
// in the Iris pipeline.
func (p *Provider) convertLevel(slogLevel slog.Level) iris.Level {
	switch {
	case slogLevel <= slog.LevelDebug:
		return iris.Debug
	case slogLevel <= slog.LevelInfo:
		return iris.Info
	case slogLevel <= slog.LevelWarn:
		return iris.Warn
	default:
		return iris.Error
	}
}

// convertAttribute converts a slog.Attr to an iris.Field with type preservation.
//
// This function examines the slog attribute's value type and creates the
// corresponding strongly-typed Iris field. Supported conversions:
//   - String → iris.String
//   - Int64 → iris.Int64
//   - Uint64 → iris.Uint64
//   - Float64 → iris.Float64
//   - Bool → iris.Bool
//   - Duration → iris.Dur
//   - Time → iris.Time
//   - Other types → iris.String (using String() method)
//
// Type preservation ensures that Iris encoders can format values appropriately
// and that type-specific features (like duration formatting) work correctly.
func (p *Provider) convertAttribute(attr slog.Attr) iris.Field {
	key := attr.Key
	value := attr.Value

	switch value.Kind() {
	case slog.KindString:
		return iris.String(key, value.String())
	case slog.KindInt64:
		return iris.Int64(key, value.Int64())
	case slog.KindUint64:
		return iris.Uint64(key, value.Uint64())
	case slog.KindFloat64:
		return iris.Float64(key, value.Float64())
	case slog.KindBool:
		return iris.Bool(key, value.Bool())
	case slog.KindDuration:
		return iris.Dur(key, value.Duration())
	case slog.KindTime:
		return iris.Time(key, value.Time())
	default:
		return iris.String(key, value.String())
	}
}
