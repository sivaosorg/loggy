// Package loggy provides a minimalist logging library with configurable severity levels,
// formatted logging, caller tracking, and thread-safe operations.
//
// Key features:
//   - Five severity levels (Debug, Info, Warn, Error, Fatal) with custom labels
//   - Customizable timestamp formatting and timezone configuration
//   - Caller source location tracking with stack depth control
//   - Thread-safe operations through locker interface compatibility
//   - Package-level default logger and configurable instances
package loggy

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// New creates a new Logger instance configured with the provided parameters and options.
// The logger's name must be formatted as ": name:" (with a colon, a space, the name, and a colon).
// The writer parameter must be non-nil, and the minLevel must be a valid severity (below DisableLogger).
//
// Parameters:
//   - name: a string identifier in the format ": name:" (e.g., ": my-service:").
//   - writer: an io.Writer where log messages will be written (e.g., os.Stdout).
//   - minLevel: the minimum Severity level to be logged; messages with a lower level are ignored.
//   - opts: a variadic slice of Option functions to customize the logger (e.g., WithUTC, WithTimeFormat).
//
// Panics:
//   - if the provided name does not follow the required format.
//   - if the writer is nil or the minLevel is invalid.
func New(name string, writer io.Writer, minLevel Severity, opts ...Option) *Logger {
	if len(name) < 3 || name[0] != ':' || name[1] != ' ' || name[len(name)-1] != ':' {
		panic("loggy: invalid name format - use ': name:'")
	}
	if writer == nil || minLevel > DisableIssuer {
		panic("loggy: invalid writer or severity level")
	}
	l := &Logger{
		name:          name,
		writer:        writer,
		minLevel:      minLevel,
		timeFormat:    "2006-01-02 15:04:05.000000",
		useUTC:        false,
		severityNames: []string{"debug:", "info:", "warn:", "error:", "fatal:"},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithTimeFormat returns an Option that sets a custom time format for log messages.
// The format should be specified using Go's reference time (Mon Jan 2 15:04:05 MST 2006).
//
// Example:
//
//	logger := New(": my-service:", os.Stdout, DebugLogger, WithTimeFormat("15:04:05"))
func WithTimeFormat(format string) Option {
	return func(l *Logger) {
		l.timeFormat = format
	}
}

// WithUTC returns an Option that configures the Logger to use UTC for timestamps if set to true,
// or the local time zone if false.
//
// Example:
//
//	logger := New(": my-service:", os.Stdout, DebugLogger, WithUTC(true))
func WithUTC(utc bool) Option {
	return func(l *Logger) {
		l.useUTC = utc
	}
}

// WithSeverityNames returns an Option that sets custom labels for the severity levels.
// The provided slice must contain exactly five labels, one for each severity level (Debug, Info, Warn, Error, Fatal).
//
// Example:
//
//	logger := New(": my-service:", os.Stdout, DebugLogger, WithSeverityNames([]string{"DBG", "INF", "WRN", "ERR", "FTL"}))
func WithSeverityNames(names []string) Option {
	return func(l *Logger) {
		if len(names) == 5 {
			l.severityNames = names
		}
	}
}

// Name returns the logger's identifier without the enclosing colons and leading space.
// For a name defined as ": my-service:", this function returns "my-service".
func (l *Logger) Name() string {
	return l.name[2 : len(l.name)-1]
}

// UpdateWriter safely updates the Logger's output destination to a new writer.
// If both the current writer and the new writer implement the locker interface but are not the same,
// the update is rejected (returns false) to avoid locking mismatches. Otherwise, the writer is updated.
// The function locks the current writer (if possible) during the update to ensure thread safety.
//
// Parameters:
//   - w: the new io.Writer to use as the logging destination.
//
// Returns:
//   - true if the writer was successfully updated.
//   - false if the update was rejected due to nil writer or incompatible locking behavior.
func (l *Logger) UpdateWriter(w io.Writer) bool {
	if w == nil {
		return false
	}
	currentLocker, hasLock := l.writer.(locker)
	newLocker, newHasLock := w.(locker)
	if hasLock && newHasLock && currentLocker != newLocker {
		return false
	}
	if hasLock {
		currentLocker.Lock()
		defer currentLocker.Unlock()
	}
	l.writer = w
	return true
}

// SetLevel changes the Logger's minimum logging severity level at runtime.
// Only messages at or above the new level will be logged.
//
// Parameters:
//   - level: the new Severity level to set. Must be a valid level (less than or equal to DisableLogger).
func (l *Logger) SetLevel(level Severity) {
	if level <= DisableIssuer {
		l.minLevel = level
	}
}

// GetLevel returns the current minimum logging severity level.
// This can be used to inspect the current filtering threshold for logging messages.
func (l *Logger) GetLevel() Severity {
	return l.minLevel
}

// Log is the core function that writes log messages to the Logger's writer if the
// message's severity is at or above the Logger's configured minimum level.
// It accepts an optional Caller argument as the first parameter to control the
// stack skip depth when capturing the caller's file and line number.
//
// Parameters:
//   - level: the Severity level of the log message.
//   - msg: one or more message components to be logged. If the first argument is of type Caller,
//     it is used to set the caller depth (number of stack frames to skip).
//
// Returns:
//   - An error if there is a failure while writing to the output; otherwise, nil.
func (l *Logger) Log(level Severity, msg ...interface{}) error {
	// Do nothing if the message severity is below the minimum level, is disabled, or no message is provided.
	if level < l.minLevel || level >= DisableIssuer || len(msg) == 0 {
		return nil
	}

	now := time.Now()
	if l.useUTC {
		now = now.UTC()
	}

	// Process the optional Caller argument (if provided as the first element).
	skip := 0
	if depth, ok := msg[0].(Caller); ok {
		skip = int(depth)
		if skip < 0 {
			skip = 0
		} else if skip > 99 {
			skip = 99
		}
		msg = msg[1:]
		if len(msg) == 0 {
			return nil
		}
	}

	// Use strings.Builder to efficiently build the complete log message.
	var b strings.Builder
	b.Grow(128) // Pre-allocate an estimated capacity to minimize allocations.

	// Compose the log prefix: timestamp, logger name, and severity label.
	b.WriteString(now.Format(l.timeFormat))
	b.WriteString(l.name)
	b.WriteString(l.severityNames[level])

	// Append caller information (file name and line number) if available.
	if _, file, line, ok := runtime.Caller(skip + 2); ok {
		b.WriteByte(' ')
		b.WriteString(filepath.Base(file))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(line))
		b.WriteByte(':')
	}

	b.WriteByte(' ')

	// Combine the log message components.
	// If there is only one message argument and it is a string, write it directly.
	if len(msg) == 1 {
		if s, ok := msg[0].(string); ok {
			b.WriteString(s)
		} else {
			b.WriteString(fmt.Sprint(msg[0]))
		}
	} else {
		// For multiple arguments, combine them using fmt.Sprint.
		b.WriteString(fmt.Sprint(msg...))
	}
	// Ensure the message ends with a newline.
	if b.Len() == 0 || b.String()[b.Len()-1] != '\n' {
		b.WriteByte('\n')
	}
	// Write the log entry to the configured writer with locking if available.
	if lock, ok := l.writer.(locker); ok {
		lock.Lock()
		defer lock.Unlock()
	}
	_, err := io.WriteString(l.writer, b.String())
	return err
}

// Debug logs a debug-level message using the Logger instance.
// An optional Caller argument may be provided as the first parameter to control the caller depth.
//
// Example:
//
//	logger.Debug("This is a debug message.")
//	logger.Debug(Caller(1), "Message from a wrapper function.")
func (l *Logger) Debug(msg ...interface{}) error {
	return l.Log(DebugIssuer, msg...)
}

// Debugf logs a formatted debug-level message using the Logger instance.
// It formats the message using the provided format string and arguments.
//
// Example:
//
//	logger.Debugf("Debug value: %v", someValue)
func (l *Logger) Debugf(format string, args ...interface{}) error {
	return l.Log(DebugIssuer, fmt.Sprintf(format, args...))
}

// Info logs an informational message using the Logger instance.
// An optional Caller argument may be provided as the first parameter to control the caller depth.
func (l *Logger) Info(msg ...interface{}) error {
	return l.Log(InfoIssuer, msg...)
}

// Infof logs a formatted informational message using the Logger instance.
// It formats the message using the provided format string and arguments.
func (l *Logger) Infof(format string, args ...interface{}) error {
	return l.Log(InfoIssuer, fmt.Sprintf(format, args...))
}

// Warn logs a warning message using the Logger instance.
// An optional Caller argument may be provided as the first parameter to control the caller depth.
func (l *Logger) Warn(msg ...interface{}) error {
	return l.Log(WarnIssuer, msg...)
}

// Warnf logs a formatted warning message using the Logger instance.
// It formats the message using the provided format string and arguments.
func (l *Logger) Warnf(format string, args ...interface{}) error {
	return l.Log(WarnIssuer, fmt.Sprintf(format, args...))
}

// Error logs an error message using the Logger instance.
// An optional Caller argument may be provided as the first parameter to control the caller depth.
func (l *Logger) Error(msg ...interface{}) error {
	return l.Log(ErrorIssuer, msg...)
}

// Errorf logs a formatted error message using the Logger instance.
// It formats the message using the provided format string and arguments.
func (l *Logger) Errorf(format string, args ...interface{}) error {
	return l.Log(ErrorIssuer, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message using the Logger instance and then triggers a panic.
// An optional Caller argument may be provided as the first parameter to control the caller depth.
// The panic message consists of the logger name and fatal severity label concatenated with any
// error string returned during the logging process.
func (l *Logger) Fatal(msg ...interface{}) error {
	err := l.Log(FatalIssuer, msg...)
	pm := l.Name() + l.severityNames[FatalIssuer]
	if err != nil {
		pm += err.Error()
	}
	panic(pm)
}

// Fatalf logs a formatted fatal message using the Logger instance and then triggers a panic.
// It formats the message using the provided format string and arguments.
// The panic message consists of the logger name and fatal severity label concatenated with any
// error string returned during the logging process.
func (l *Logger) Fatalf(format string, args ...interface{}) error {
	err := l.Log(FatalIssuer, fmt.Sprintf(format, args...))
	pm := l.Name() + l.severityNames[FatalIssuer]
	if err != nil {
		pm += err.Error()
	}
	panic(pm)
}

// Debug logs a debug-level message using the package-level Default logger.
// An optional Caller argument may be provided as the first parameter.
func Debug(msg ...interface{}) error {
	return Default.Log(DebugIssuer, msg...)
}

// Debugf logs a formatted debug-level message using the package-level Default logger.
func Debugf(format string, args ...interface{}) error {
	return Default.Log(DebugIssuer, fmt.Sprintf(format, args...))
}

// Info logs an informational message using the package-level Default logger.
// An optional Caller argument may be provided as the first parameter.
func Info(msg ...interface{}) error {
	return Default.Log(InfoIssuer, msg...)
}

// Infof logs a formatted informational message using the package-level Default logger.
func Infof(format string, args ...interface{}) error {
	return Default.Log(InfoIssuer, fmt.Sprintf(format, args...))
}

// Warn logs a warning message using the package-level Default logger.
// An optional Caller argument may be provided as the first parameter.
func Warn(msg ...interface{}) error {
	return Default.Log(WarnIssuer, msg...)
}

// Warnf logs a formatted warning message using the package-level Default logger.
func Warnf(format string, args ...interface{}) error {
	return Default.Log(WarnIssuer, fmt.Sprintf(format, args...))
}

// Error logs an error message using the package-level Default logger.
// An optional Caller argument may be provided as the first parameter.
func Error(msg ...interface{}) error {
	return Default.Log(ErrorIssuer, msg...)
}

// Errorf logs a formatted error message using the package-level Default logger.
func Errorf(format string, args ...interface{}) error {
	return Default.Log(ErrorIssuer, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message using the package-level Default logger and then triggers a panic.
// An optional Caller argument may be provided as the first parameter.
func Fatal(msg ...interface{}) error {
	err := Default.Log(FatalIssuer, msg...)
	pm := Default.Name() + Default.severityNames[FatalIssuer]
	if err != nil {
		pm += err.Error()
	}
	panic(pm)
}

// Fatalf logs a formatted fatal message using the package-level Default logger and then triggers a panic.
func Fatalf(format string, args ...interface{}) error {
	err := Default.Log(FatalIssuer, fmt.Sprintf(format, args...))
	pm := Default.Name() + Default.severityNames[FatalIssuer]
	if err != nil {
		pm += err.Error()
	}
	panic(pm)
}
