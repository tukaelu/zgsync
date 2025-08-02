package testhelper

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureStdout captures stdout during function execution and returns the output
// This function is safe for parallel test execution
func CaptureStdout(t *testing.T, fn func() error) (string, error) {
	// Save original stdout
	originalStdout := os.Stdout
	
	// Create pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	
	// Replace stdout
	os.Stdout = w
	
	// Execute function
	funcErr := fn()
	
	// Close writer and restore stdout
	w.Close()
	os.Stdout = originalStdout
	
	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	
	return buf.String(), funcErr
}

// Constants for common test values to reduce magic numbers
const (
	TestSectionID         = 123
	TestArticleID         = 456
	TestPermissionGroupID = 789
	TestUserSegmentID     = 999
)

// TestLocales contains common locale values used in tests
var TestLocales = struct {
	Japanese string
	English  string
	French   string
}{
	Japanese: "ja",
	English:  "en_us", 
	French:   "fr",
}