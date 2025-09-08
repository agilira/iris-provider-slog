# Iris Provider for Slog
### an AGILira library

[![CI/CD](https://github.com/agilira/iris-provider-slog/workflows/CI%2FCD/badge.svg)](https://github.com/agilira/iris-provider-slog/actions/workflows/ci.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/iris-provider-slog/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/iris-provider-slog)](https://goreportcard.com/report/github.com/agilira/iris-provider-slog)
[![Made For Iris](https://img.shields.io/badge/Made_for-Iris-pink)](https://github.com/agilira/iris)

External provider for integrating Go's standard `log/slog` package with [Iris](https://github.com/agilira/iris) logging framework.

## Features

- **Zero Code Changes**: Use existing slog code unchanged
- **Performance**: Benefit from Iris's 31ns/op logging performance  
- **Feature Inheritance**: Automatic OpenTelemetry, writers, security features
- **Drop-in Replacement**: Simply replace slog.Handler with Iris provider

## Installation

```bash
go get github.com/agilira/iris-provider-slog
```

## Usage

### Basic Integration

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/agilira/iris"
    slogprovider "github.com/agilira/iris-provider-slog"
)

func main() {
    // Create slog provider
    provider := slogprovider.New(1000)
    defer provider.Close()
    
    // Create Iris logger with provider
    readers := []iris.SyncReader{provider}
    logger, _ := iris.NewReaderLogger(iris.Config{
        Output:  iris.WrapWriter(os.Stdout),
        Encoder: iris.NewJSONEncoder(),
        Level:   iris.Info,
    }, readers)
    defer logger.Close()
    
    logger.Start()
    
    // Use slog normally - but get Iris performance!
    slogger := slog.New(provider)
    slogger.Info("User login", "user_id", "12345")
}
```

### Advanced Features

```go
// For advanced features like Loki integration, install the writer module:
// go get github.com/agilira/iris-writer-loki

import loki "github.com/agilira/iris-writer-loki"

// Create Loki writer
lokiWriter, err := loki.NewWriter(loki.Config{
    Endpoint: "http://loki:3100/loki/api/v1/push",
    Labels: map[string]string{
        "service": "my-app",
        "source":  "slog",
    },
})
if err != nil {
    panic(err)
}
defer lokiWriter.Close()

// Create Iris logger with all advanced features
logger, _ := iris.NewReaderLogger(iris.Config{
    Output: iris.MultiWriter(
        iris.WrapWriter(os.Stdout),
        lokiWriter, // Loki integration via external module
    ),
    Encoder: iris.NewJSONEncoder(),
    Level:   iris.Debug,
}, readers,
    iris.WithOTel(),    // OpenTelemetry integration
    iris.WithCaller(),  // Caller information
)

// Now slog gets ALL Iris features automatically:
// ✅ OpenTelemetry trace correlation
// ✅ Grafana Loki batching  
// ✅ Automatic secret redaction
// ✅ High-performance ring buffer
slogger.Info("Payment", "amount", 100, "api_key", "secret") // api_key redacted
```

## Examples

See the [examples/basic](./examples/basic/main.go) directory for a complete working example demonstrating all features.

Run the basic example:
```bash
go run examples/basic/main.go
```

## Performance

- **slog Handle**: ~56 ns/op
- **Record Conversion**: ~745 ns/op  
- **Overall**: 20x+ faster than standard slog

## Architecture

This provider implements the `iris.SyncReader` interface, allowing slog records to be processed by Iris's high-performance pipeline:

```
slog.Logger → SlogProvider → iris.SyncReader → Iris Ring Buffer → Features
```

## License

iris-provider-slog is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

iris-provider-slog • an AGILira library
