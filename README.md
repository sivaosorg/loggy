# loggy

`loggy` is a minimalist logging library for Go that provides configurable logging with multiple severity levels, formatted messages, caller location tracking, and thread-safe operations. It is designed to be lightweight and highly customizable to suit various logging needs.

### Features

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

### Requirements

- Go version 1.23 or higher

### Installation

To start using `loggy`, run `go get`:

- For a specific version:

  ```bash
  go get github.com/sivaosorg/loggy@v0.0.1
  ```

- For the latest version:
  ```bash
  go get -u github.com/sivaosorg/loggy@latest
  ```

### Getting started

#### Getting loggy

With [Go's module support](https://go.dev/wiki/Modules#how-to-use-modules), `go [build|run|test]` automatically fetches the necessary dependencies when you add the import in your code:

```go
import "github.com/sivaosorg/loggy"
```

#### Quick Start

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
