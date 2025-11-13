# Log Package

`log` is a Go package that provides a simplified and opinionated interface for structured logging, built on top of the high-performance [zap](https://github.com/uber-go/zap) logger. It offers a globally configured, easy-to-use logger that can be safely used across your entire application.

## Core Features

- **Global Singleton Logger**: A single logger instance is used throughout the application, initialized once to ensure consistency and performance.
- **Simplified Configuration**: Easily configure the logger for different environments (development vs. production) and set log levels with a clean, functional options pattern.
- **Structured and Formatted Logging**: Supports both highly performant structured logging (e.g., `log.Info("message", log.String("key", "value"))`) and convenient formatted logging (e.g., `log.Infof("user %s logged in", username)`).
- **Convenient Field Constructors**: Provides simple wrappers (e.g., `log.String`, `log.Int`, `log.Err`) for zap's structured fields, making logging code more readable.
- **Environment-Aware Output**: Automatically provides human-readable, colored output in development mode and structured, efficient JSON output in production.

## How to Use

### 1. Configure the Logger (Optional)

In your application's `main` or `init` function, you can configure the logger. This step is optional. If you don't configure it, it will run with sensible defaults (development mode, debug level).

**Configuration should only be done once.**

```go
package main

import (
	"os"
	"github.com/94peter/vulpes/log"
)

func main() {
    // Example: Configure for a production environment.
    // The environment could be determined by an environment variable.
    if os.Getenv("APP_ENV") == "production" {
        log.SetConfig(
            log.WithDev(false),      // Disable development mode for JSON output.
            log.WithLevel("info"),   // Set the log level to info.
        )
    }

    // ... rest of your application logic
}
```

### 2. Log Messages

Once configured (or using defaults), you can call the logging functions from anywhere in your application.

#### Structured Logging (Recommended)

Structured logging is the preferred method, especially in production, as it makes logs machine-readable and easier to query.

```go
import "github.com/94peter/vulpes/log"

func processRequest(requestID string, userID int) error {
    log.Info("Processing request",
        log.String("request_id", requestID),
        log.Int("user_id", userID),
    )

    err := doSomething()
    if err != nil {
        // The Err field is a convenient way to log errors.
        log.Error("Failed to do something",
            log.String("request_id", requestID),
            log.Err(err),
        )
        return err
    }

    log.Debug("Request processed successfully", log.String("request_id", requestID))
    return nil
}
```

#### Formatted Logging

For simple messages or during early development, `...f` style functions can be more convenient.

```go
import "github.com/94peter/vulpes/log"

func greet(name string) {
    log.Infof("Hello, %s! Welcome to the application.", name)
}
```

## API Reference

### Configuration

- `SetConfig(opts ...Option)`: Configures the global logger. Can only be effectively called once.
- `WithDev(bool)`: An option to set development mode.
- `WithLevel(string)`: An option to set the log level (e.g., `"debug"`, `"info"`, `"warn"`, `"error"`).

### Logging Methods

- `Debug(msg string, fields ...Field)`
- `Info(msg string, fields ...Field)`
- `Warn(msg string, fields ...Field)`
- `Error(msg string, fields ...Field)`
- `Panic(msg string, fields ...Field)`
- `Fatal(msg string, fields ...Field)`

### Formatted Logging Methods

- `Debugf(format string, a ...interface{})`
- `Infof(format string, a ...interface{})`
- `Warnf(format string, a ...interface{})`
- `Errorf(format string, a ...interface{})`
- `Fatalf(format string, a ...interface{})`

### Field Constructors

- `String(key, val string) Field`
- `Int(key, val int) Field`
- `Int64(key, val int64) Field`
- `Bool(key, val bool) Field`
- `Time(key, val time.Time) Field`
- `Duration(key, val time.Duration) Field`
- `Err(err error) Field`
- `Any(key, val interface{}) Field`
- ... and many more for other primitive types.
