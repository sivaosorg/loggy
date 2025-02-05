package loggy

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// dummyLocker is an io.Writer that implements the locker interface.
// It records the writes in a bytes.Buffer.
type dummyLocker struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (d *dummyLocker) Write(p []byte) (int, error) {
	return d.buf.Write(p)
}

func (d *dummyLocker) Lock() {
	d.mu.Lock()
}

func (d *dummyLocker) Unlock() {
	d.mu.Unlock()
}

// TestNewValid verifies that a Logger created with a valid name and writer
// does not panic and returns the correct name.
func TestNewValid(t *testing.T) {
	// valid name format: ": name:"
	name := ": test-service:"
	buf := new(bytes.Buffer)
	logger := New(name, buf, DebugIssuer)
	if logger.Name() != "test-service" {
		t.Errorf("Expected logger name 'test-service', got '%s'", logger.Name())
	}
}

// TestNewInvalidName verifies that New panics if the name format is invalid.
func TestNewInvalidName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid logger name, but did not panic")
		}
	}()
	// Invalid name: missing leading colon and space.
	_ = New("test-service", new(bytes.Buffer), DebugIssuer)
}

// TestNewInvalidWriter verifies that New panics if the writer is nil.
func TestNewInvalidWriter(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil writer, but did not panic")
		}
	}()
	_ = New(": test-service:", nil, DebugIssuer)
}

// TestLogSeverityFiltering ensures that messages below the minimum level are not logged.
func TestLogSeverityFiltering(t *testing.T) {
	buf := new(bytes.Buffer)
	// Set minimum level to InfoIssuer. Debug messages should be filtered out.
	logger := New(": test-service:", buf, InfoIssuer)
	if err := logger.Debug("this debug message should be filtered out"); err != nil {
		t.Errorf("Unexpected error from Debug: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected no output for debug message below InfoIssuer, got: %s", buf.String())
	}
	// An Info message should appear.
	if err := logger.Info("this info message should appear"); err != nil {
		t.Errorf("Unexpected error from Info: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Expected output for info message, but got none")
	}
}

// TestLogOutput verifies that a log message is properly formatted and contains expected substrings.
func TestLogOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := New(": test-service:", buf, DebugIssuer)
	message := "hello, world"
	if err := logger.Info(message); err != nil {
		t.Errorf("Unexpected error from Info: %v", err)
	}
	output := buf.String()

	// Check that the output starts with a timestamp. We check that the first character is a digit.
	if len(output) < 1 || output[0] < '0' || output[0] > '9' {
		t.Errorf("Expected log output to start with a timestamp, got: %s", output)
	}

	// Check that the logger name and severity label are present.
	if !strings.Contains(output, ": test-service:") {
		t.Errorf("Expected logger name ': test-service:' in output, got: %s", output)
	}
	if !strings.Contains(output, "info:") {
		t.Errorf("Expected severity label 'info:' in output, got: %s", output)
	}
	// Check that the message is included.
	if !strings.Contains(output, message) {
		t.Errorf("Expected message '%s' in output, got: %s", message, output)
	}
	// Check that the output ends with a newline.
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("Expected log output to end with a newline, got: %q", output[len(output)-1])
	}
}

// TestUpdateWriter tests the UpdateWriter method including the locking behavior.
func TestUpdateWriter(t *testing.T) {
	t.Run("update to non-locking writer", func(t *testing.T) {
		// Start with a logger whose writer is a dummyLocker (implements locker).
		dl := &dummyLocker{}
		logger := New(": test-service:", dl, DebugIssuer)

		// Update to a writer without locking: should succeed.
		buf := new(bytes.Buffer)
		if ok := logger.UpdateWriter(buf); !ok {
			t.Error("Expected UpdateWriter to succeed with non-locking writer")
		}
	})

	t.Run("update to different locker writer", func(t *testing.T) {
		// Start with a logger whose writer is a dummyLocker.
		dl1 := &dummyLocker{}
		logger := New(": test-service:", dl1, DebugIssuer)

		// Attempt to update to a different dummyLocker (also implements locker).
		dl2 := &dummyLocker{}
		if ok := logger.UpdateWriter(dl2); ok {
			t.Error("Expected UpdateWriter to reject update with different locker writer")
		}
	})

	t.Run("update to nil writer", func(t *testing.T) {
		// Start with a logger whose writer is a dummyLocker.
		dl := &dummyLocker{}
		logger := New(": test-service:", dl, DebugIssuer)

		// Update to nil should be rejected.
		if ok := logger.UpdateWriter(nil); ok {
			t.Error("Expected UpdateWriter to reject nil writer")
		}
	})
}

// TestSetAndGetLevel verifies that SetLevel and GetLevel work as expected.
func TestSetAndGetLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := New(": test-service:", buf, DebugIssuer)
	// Set level to WarnIssuer.
	logger.SetLevel(WarnIssuer)
	if got := logger.GetLevel(); got != WarnIssuer {
		t.Errorf("Expected level %d, got %d", WarnIssuer, got)
	}
	// Attempt to set an invalid level (greater than DisableIssuer) should be ignored.
	logger.SetLevel(DisableIssuer + 1)
	if got := logger.GetLevel(); got != WarnIssuer {
		t.Errorf("Expected level to remain %d after invalid update, got %d", WarnIssuer, got)
	}
}

// TestCaller ensures that the Caller argument is correctly processed.
// It uses runtime.Caller to get the expected file name.
func TestCaller(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := New(": test-service:", buf, DebugIssuer)
	// We know that the log call is here, so runtime.Caller(0) would refer to this function.
	// Use Caller(0) so that the runtime.Caller(skip+2) in Log points to this function.
	if err := logger.Info(Caller(0), "testing caller"); err != nil {
		t.Errorf("Unexpected error from Info with Caller: %v", err)
	}
	output := buf.String()
	// Check that the output contains this file's base name.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	base := filepath.Base(file)
	if !strings.Contains(output, base) {
		t.Errorf("Expected output to contain caller file name %q, got: %s", base, output)
	}
}

// TestFatal verifies that Fatal logs the message and then panics.
func TestFatal(t *testing.T) {
	buf := new(bytes.Buffer)
	// Use a custom logger to capture output.
	logger := New(": test-service:", buf, DebugIssuer)
	defer func() {
		if r := recover(); r != nil {
			// Check that the panic message contains the logger name and fatal severity label.
			panicMsg, ok := r.(string)
			if !ok {
				t.Errorf("Expected panic message to be a string, got %T", r)
			}
			if !strings.Contains(panicMsg, "test-service") {
				t.Errorf("Expected panic message to contain 'test-service', got: %s", panicMsg)
			}
			if !strings.Contains(panicMsg, "fatal:") {
				t.Errorf("Expected panic message to contain 'fatal:', got: %s", panicMsg)
			}
		} else {
			t.Error("Expected Fatal to panic, but it did not")
		}
	}()
	// This call should panic.
	_ = logger.Fatal("fatal error occurred")
}

// TestFormattedLogging tests the formatted logging functions (Debugf, Infof, etc).
func TestFormattedLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := New(": test-service:", buf, DebugIssuer)
	testVal := 42
	if err := logger.Debugf("debug value: %d", testVal); err != nil {
		t.Errorf("Unexpected error from Debugf: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "debug value: 42") {
		t.Errorf("Expected formatted message to contain 'debug value: 42', got: %s", output)
	}
}

// TestPackageLevelFunctions tests the package-level default logger functions.
// Note: Because Default is a global logger, these tests may interact with other tests if run concurrently.
func TestPackageLevelFunctions(t *testing.T) {
	// Redirect Default logger's output to a buffer for testing.
	buf := new(bytes.Buffer)
	// Save the original writer so we can restore it later.
	origWriter := Default.writer
	defer func() {
		Default.writer = origWriter
	}()
	Default.writer = buf

	// Test Info function.
	Info("package level info")
	output := buf.String()
	if !strings.Contains(output, "info:") {
		t.Errorf("Expected output to contain 'info:' for package-level Info, got: %s", output)
	}

	// Clear buffer and test Infof.
	buf.Reset()
	Infof("package infof: %d", 100)
	output = buf.String()
	if !strings.Contains(output, "package infof: 100") {
		t.Errorf("Expected output to contain 'package infof: 100', got: %s", output)
	}
}

// TestTimeFormatAndUTC verifies that WithTimeFormat and WithUTC options work.
func TestTimeFormatAndUTC(t *testing.T) {
	buf := new(bytes.Buffer)
	// Use a custom time format.
	customFormat := "15:04:05"
	logger := New(": test-service:", buf, DebugIssuer, WithTimeFormat(customFormat), WithUTC(true))
	// Log a message.
	if err := logger.Info("time test"); err != nil {
		t.Errorf("Unexpected error from Info: %v", err)
	}
	output := buf.String()
	// Extract the timestamp substring which is the first len(customFormat) characters.
	if len(output) < len(customFormat) {
		t.Fatalf("Unexpected log format: %s", output)
	}
	timestamp := output[:len(customFormat)]
	// Parse the timestamp using the custom format.
	_, err := time.Parse(customFormat, timestamp)
	if err != nil {
		t.Errorf("Timestamp %q does not match format %q: %v", timestamp, customFormat, err)
	}
}
