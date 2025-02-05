# loggy

`loggy` is a minimalist logging library for Go that provides configurable logging with multiple severity levels, formatted messages, caller location tracking, and thread-safe operations. It is designed to be lightweight and highly customizable to suit various logging needs.

## Features

- **Severity Levels:** Supports five levels of logging:
  - `Debug`
  - `Info`
  - `Warn`
  - `Error`
  - `Fatal` (triggers a panic)
- **Formatted Logging:** Use both direct and formatted log methods.
- **Customization:** Easily configure output time formats, logger names, and severity labels.
- **Caller Location:** Optionally include caller information (file and line number) in log messages.
- **Thread-Safe:** Supports concurrent logging by locking the writer if it implements a locker interface.
- **Multiple Logger Instances:** Create package-specific logger instances or use the provided default logger.

## Requirements

- Go version 1.23 or higher

## Installation

To start using `loggy`, run `go get`:

- For a specific version:

  ```bash
  go get github.com/sivaosorg/loggy@v0.0.1
  ```

- For the latest version:
  ```bash
  go get -u github.com/sivaosorg/loggy@latest
  ```

## Getting started

### Getting loggy

With [Go's module support](https://go.dev/wiki/Modules#how-to-use-modules), `go [build|run|test]` automatically fetches the necessary dependencies when you add the import in your code:

```go
import "github.com/sivaosorg/loggy"
```

### Quick Start

Here is a simple example that demonstrates how to use `loggy`:

```go
package main

import (
	"os"

	"github.com/sivaosorg/loggy"
)

func main() {
	// Create a new logger instance.
	// The logger name must be in the format ": name:".
	logger := loggy.New(": my-service:", os.Stdout, loggy.DebugIssuer,
		loggy.WithUTC(true),
		loggy.WithTimeFormat("2006-01-02 15:04:05.000000"),
		loggy.WithSeverityNames([]string{"DEBUG: ", "INFO: ", "WARN: ", "ERROR: ", "FATAL: "}),
	)

	// Log a debug message.
	logger.Debug("This is a debug message.")

	// Log an informational message.
	logger.Info("Service started successfully.")

	// Log a warning message with formatted output.
	logger.Warnf("Cache miss for key: %s", "user123")

	// Log an error message.
	logger.Error("Failed to connect to database.")

	// Log a fatal error (this will panic).
	// logger.Fatal("Critical error, shutting down!")
}
```

### Usage

#### Creating a Logger

Use the `New` function to create a custom logger instance:

```go
logger := loggy.New(": my-service:", os.Stdout, loggy.DebugIssuer,
		loggy.WithUTC(true),
		loggy.WithTimeFormat("2006-01-02 15:04:05.000000"),
		loggy.WithSeverityNames([]string{"DEBUG: ", "INFO: ", "WARN: ", "ERROR: ", "FATAL: "}),
	)
```

Parameters:

- **name**: Must be in the format `": name:"`. For example, `": my-service:"`.
- **writer**: An `io.Writer` where log messages will be sent (e.g., `os.Stdout`).
- **minLevel**: The minimum severity level to log. Messages below this level are ignored.
- **Options**: Additional configuration options provided via option functions (e.g., `WithUTC`, `WithTimeFormat`, `WithSeverityNames`).

#### Logging Messages

Each logging method supports an optional caller depth argument (of type `Caller`) to specify how many stack frames to skip when reporting the caller's file and line number. If no caller is needed, simply omit it.

- **Direct Logging**:

```go
logger.Debug("This is a debug message.")
logger.Info("Service started successfully.")
logger.Warn("This is a warning message.")
logger.Error("An error occurred.")
// logger.Fatal("A fatal error occurred.") // Will panic after logging.
```

- **Formatted Logging:**

```go
logger.Debugf("Debug info: %v", someValue)
logger.Infof("User %s logged in", username)
logger.Warnf("Cache miss for key: %s", cacheKey)
logger.Errorf("Error processing request: %s", err)
// logger.Fatalf("Fatal error: %s", err) // Will panic after logging.
```

#### Caller Depth Control

To include a custom caller depth (e.g., when wrapping log calls in your own functions), provide a `Caller` value as the first argument:

```go
// Caller depth set to 1.
logger.Info(loggy.Caller(1), "Called from a wrapper function")
```

#### Using the Default Logger

`loggy` also provides a package-level default logger. You can use it directly with convenience functions:

```go
package main

import (
    "github.com/sivaosorg/loggy"
)

func main() {
    loggy.Debug("Using default logger for debug")
    loggy.Infof("Hello, %s!", "world")
    // loggy.Fatal("This is a fatal message") // Will panic.
}
```

#### Updating the Writer

To safely change the output destination of a logger, use the `UpdateWriter` method:

```go
if ok := logger.UpdateWriter(newWriter); !ok {
    logger.Error("Writer update failed: incompatible locker interface")
}
```

#### Setting the Logging Level

Adjust the minimum severity level at runtime:

```go
logger.SetLevel(loggy.InfoIssuer)
```

And query the current level with:

```go
currentLevel := logger.GetLevel()
```
