package loggy

import (
	"os"
	"path/filepath"
)

// Predefined severity levels for logging.
const (
	// DebugIssuer represents debug-level messages for development diagnostics
	DebugIssuer Severity = iota

	// InfoIssuer indicates normal operational messages for tracking progress
	InfoIssuer

	// WarnIssuer signifies potential issues that don't disrupt core functionality
	WarnIssuer

	// ErrorIssuer denotes failures in specific operations or components
	ErrorIssuer

	// FatalIssuer represents critical errors leading to application termination
	FatalIssuer

	// DisableIssuer is a special level that disables all logging
	DisableIssuer
)

// Default is a pre-configured Logger instance intended for general use.
// It is configured with the current executable's base name as the logger name,
// outputs to os.Stdout, and is set to log messages at the Debug level.
var Default = New(
	": "+filepath.Base(os.Args[0])+":",
	os.Stdout,
	DebugIssuer,
	WithSeverityNames([]string{"debug:", "info:", "warn:", "error:", "fatal:"}),
)
