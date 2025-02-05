package loggy

import "io"

// Severity defines the logging severity level as an unsigned 32-bit integer.
// Lower values indicate higher priority messages.
type Severity uint32

// Logger represents a logging instance with its configuration settings. It includes
// the logger's identifier, output destination, severity filtering level, time format,
// timezone configuration, and custom severity names.
type Logger struct {
	name          string    // Logger identifier in the format ": name:".
	writer        io.Writer // Destination for log output (e.g., os.Stdout).
	minLevel      Severity  // Minimum severity level to log; lower levels are ignored.
	timeFormat    string    // Format for timestamps (Go reference time format).
	useUTC        bool      // If true, log timestamps are in UTC; otherwise, local time.
	severityNames []string  // Custom labels for each severity level.
}

// Option defines a functional option for configuring a Logger instance during creation.
// Each Option is a function that accepts a pointer to a Logger and modifies its configuration.
type Option func(*Logger)

// Caller is a type alias for specifying the caller stack skip depth.
// It allows the developer to indicate how many stack frames to skip when reporting
// the source location (file and line number) of the log call.
type Caller int

// locker is an interface that defines basic locking operations.
// If an io.Writer implements this interface, it can be locked during writes to ensure thread safety.
type locker interface {
	Lock()
	Unlock()
}
