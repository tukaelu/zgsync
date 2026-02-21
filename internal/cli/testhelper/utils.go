package testhelper

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CaptureStdout captures stdout during function execution and returns the output.
// WARNING: This function modifies global os.Stdout and is NOT safe for parallel test execution.
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
	_ = w.Close()
	os.Stdout = originalStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	return buf.String(), funcErr
}

// AssertErrorContainsKeyword checks that the error message contains at least one of the given keywords.
// This is useful for validating that error context is preserved when errors propagate through layers.
func AssertErrorContainsKeyword(t *testing.T, err error, keywords []string) {
	t.Helper()
	if len(keywords) == 0 {
		return
	}
	errorMsg := err.Error()
	for _, keyword := range keywords {
		if strings.Contains(errorMsg, keyword) {
			return
		}
	}
	t.Errorf("Error message %q does not contain any of keywords: %v", errorMsg, keywords)
}

// CreateTestFile creates a test file in the given directory and returns its path.
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// TestTranslationContent is a common Markdown frontmatter used across CLI tests for translation push.
const TestTranslationContent = `---
locale: ja
title: "Test Translation"
source_id: 123
---
# Test Content`

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
