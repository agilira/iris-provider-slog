// integration_test.go: Full integration test with NewReaderLogger
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package slogprovider

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/agilira/iris"
)

// bufferedWriter captures output for testing
type bufferedWriter struct {
	data []byte
}

func (b *bufferedWriter) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *bufferedWriter) Sync() error {
	return nil
}

func (b *bufferedWriter) String() string {
	return string(b.data)
}

func TestFullIntegrationWithNewReaderLogger(t *testing.T) {
	// Create provider
	provider := New(100)
	defer provider.Close() //nolint:errcheck

	// Create buffered output
	buf := &bufferedWriter{}

	// Create ReaderLogger with provider
	readers := []iris.SyncReader{provider}
	logger, err := iris.NewReaderLogger(iris.Config{
		Output:  buf,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Debug,
	}, readers)
	if err != nil {
		t.Fatalf("Failed to create ReaderLogger: %v", err)
	}
	defer func() { _ = logger.Close() }() // Ignore error in test cleanup

	// Start the logger
	logger.Start()

	// Create slog logger using our provider
	slogger := slog.New(provider)

	// Log various levels and types
	slogger.Debug("Debug message", "key", "debug_value")
	slogger.Info("Info message", "user_id", "12345", "action", "login")
	slogger.Warn("Warning message", "component", "auth", "retry_count", 3)
	slogger.Error("Error message", "error_code", "AUTH_FAILED", "duration", 150)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Sync to ensure all records are written
	err = logger.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	output := buf.String()
	t.Logf("Output: %s", output)

	// Verify all messages were processed
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	// Verify content
	testCases := []struct {
		level   string
		message string
		key     string
		value   string
	}{
		{"debug", "Debug message", "key", "debug_value"},
		{"info", "Info message", "user_id", "12345"},
		{"warn", "Warning message", "component", "auth"},
		{"error", "Error message", "error_code", "AUTH_FAILED"},
	}

	for i, tc := range testCases {
		if i >= len(lines) {
			t.Errorf("Missing log line %d", i)
			continue
		}

		line := lines[i]
		if !strings.Contains(line, `"level":"`+tc.level+`"`) {
			t.Errorf("Line %d: expected level %s, got: %s", i, tc.level, line)
		}
		if !strings.Contains(line, `"msg":"`+tc.message+`"`) {
			t.Errorf("Line %d: expected message %s, got: %s", i, tc.message, line)
		}
		if !strings.Contains(line, `"`+tc.key+`":"`+tc.value+`"`) {
			t.Errorf("Line %d: expected field %s=%s, got: %s", i, tc.key, tc.value, line)
		}
	}
}

func TestProviderWithMultipleReaders(t *testing.T) {
	// Create multiple providers
	provider1 := New(50)
	defer func() { _ = provider1.Close() }() // Ignore error in test cleanup

	provider2 := New(50)
	defer func() { _ = provider2.Close() }() // Ignore error in test cleanup

	// Create buffered output
	buf := &bufferedWriter{}

	// Create ReaderLogger with multiple providers
	readers := []iris.SyncReader{provider1, provider2}
	logger, err := iris.NewReaderLogger(iris.Config{
		Output:  buf,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	}, readers)
	if err != nil {
		t.Fatalf("Failed to create ReaderLogger: %v", err)
	}
	defer func() { _ = logger.Close() }() // Ignore error in test cleanup

	logger.Start()

	// Create slog loggers for each provider
	slogger1 := slog.New(provider1)
	slogger2 := slog.New(provider2)

	// Log from both
	slogger1.Info("Message from logger 1", "source", "logger1")
	slogger2.Info("Message from logger 2", "source", "logger2")

	// Give time for processing
	time.Sleep(100 * time.Millisecond)
	err = logger.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Verify both messages are present (order may vary)
	fullOutput := output
	if !strings.Contains(fullOutput, "Message from logger 1") {
		t.Error("Missing message from logger 1")
	}
	if !strings.Contains(fullOutput, "Message from logger 2") {
		t.Error("Missing message from logger 2")
	}
	if !strings.Contains(fullOutput, `"source":"logger1"`) {
		t.Error("Missing source field from logger 1")
	}
	if !strings.Contains(fullOutput, `"source":"logger2"`) {
		t.Error("Missing source field from logger 2")
	}
}

func TestProviderPerformanceBasic(t *testing.T) {
	provider := New(1000)
	defer func() { _ = provider.Close() }() // Ignore error in test cleanup

	// Measure provider Handle performance
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.Add("key", "value")

	ctx := context.Background()

	// Warmup
	for i := 0; i < 100; i++ {
		_ = provider.Handle(ctx, record) // Ignore error in warmup
	}

	start := time.Now()
	n := 1000
	for i := 0; i < n; i++ {
		err := provider.Handle(ctx, record)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}
	}
	duration := time.Since(start)

	nsPerOp := duration.Nanoseconds() / int64(n)
	t.Logf("Handle performance: %d ns/op (%d ops in %v)", nsPerOp, n, duration)

	// Should be well under 500ns/op for simple handling (but allow more with race detector)
	maxNsPerOp := 500
	if testing.Short() {
		maxNsPerOp = 1000 // More lenient for race detector
	}
	if nsPerOp > int64(maxNsPerOp) {
		t.Errorf("Handle too slow: %d ns/op (expected < %d)", nsPerOp, maxNsPerOp)
	}
}
