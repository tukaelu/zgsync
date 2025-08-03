package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// FileHelper provides utilities for file operations in tests
type FileHelper struct {
	t       *testing.T
	tempDir string
}

// NewFileHelper creates a new FileHelper instance with a temporary directory
func NewFileHelper(t *testing.T) *FileHelper {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "zgsync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	return &FileHelper{t: t, tempDir: tempDir}
}

// TempDir returns the temporary directory path
func (fh *FileHelper) TempDir() string {
	return fh.tempDir
}

// CreateFile creates a file with the given content in the temp directory
func (fh *FileHelper) CreateFile(filename, content string) string {
	fh.t.Helper()
	filePath := filepath.Join(fh.tempDir, filename)

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fh.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		fh.t.Fatalf("Failed to create file %s: %v", filePath, err)
	}

	return filePath
}

// ReadFile reads the content of a file
func (fh *FileHelper) ReadFile(filename string) string {
	fh.t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		fh.t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return string(content)
}

// FileExists checks if a file exists
func (fh *FileHelper) FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// AssertFileExists asserts that a file exists
func (fh *FileHelper) AssertFileExists(filename string) {
	fh.t.Helper()
	if !fh.FileExists(filename) {
		fh.t.Errorf("Expected file %s to exist", filename)
	}
}

// AssertFileNotExists asserts that a file does not exist
func (fh *FileHelper) AssertFileNotExists(filename string) {
	fh.t.Helper()
	if fh.FileExists(filename) {
		fh.t.Errorf("Expected file %s to not exist", filename)
	}
}

// AssertFileContent asserts that a file contains specific content
func (fh *FileHelper) AssertFileContent(filename, expectedContent string) {
	fh.t.Helper()
	fh.AssertFileExists(filename)
	content := fh.ReadFile(filename)

	ah := NewAssertionHelper(fh.t)
	ah.Equal(expectedContent, content, "file content")
}

// AssertFileContains asserts that a file contains a specific substring
func (fh *FileHelper) AssertFileContains(filename, expectedSubstring string) {
	fh.t.Helper()
	fh.AssertFileExists(filename)
	content := fh.ReadFile(filename)

	ah := NewAssertionHelper(fh.t)
	ah.Contains(content, expectedSubstring, "file content")
}

// CreateTestDataFile creates a test data file with frontmatter structure
func (fh *FileHelper) CreateTestDataFile(filename, frontmatter, body string) string {
	content := "---\n" + frontmatter + "\n---\n" + body
	return fh.CreateFile(filename, content)
}

// CreateInvalidYAMLFile creates a file with invalid YAML for testing error cases
func (fh *FileHelper) CreateInvalidYAMLFile(filename string) string {
	invalidYAML := `invalid: yaml: structure
  malformed
    frontmatter`
	return fh.CreateFile(filename, invalidYAML)
}

// CreateConfigFile creates a configuration file for testing
func (fh *FileHelper) CreateConfigFile(filename string, config map[string]interface{}) string {
	content := ""
	for key, value := range config {
		switch v := value.(type) {
		case string:
			content += fmt.Sprintf("%s: %s\n", key, v)
		case int:
			content += fmt.Sprintf("%s: %d\n", key, v)
		case bool:
			content += fmt.Sprintf("%s: %t\n", key, v)
		}
	}
	return fh.CreateFile(filename, content)
}

// CreateArticleTestFile creates a test article file with proper structure
func (fh *FileHelper) CreateArticleTestFile(filename string, locale string, permissionGroupID int, title string) string {
	frontmatter := fmt.Sprintf(`locale: %s
permission_group_id: %d
title: %s`, locale, permissionGroupID, title)
	return fh.CreateTestDataFile(filename, frontmatter, "# Article Body")
}

// CreateTranslationTestFile creates a test translation file with proper structure
func (fh *FileHelper) CreateTranslationTestFile(filename string, locale string, title string, sourceID int, body string) string {
	frontmatter := fmt.Sprintf(`locale: %s
title: %s
source_id: %d`, locale, title, sourceID)
	return fh.CreateTestDataFile(filename, frontmatter, body)
}
