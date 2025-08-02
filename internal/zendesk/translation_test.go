package zendesk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslationFromFile(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Translation
	}{
		{
			"testdata/translation-ja.md",
			Translation{
				Locale:   "ja",
				Title:    "zgsyncの使い方",
				SourceID: 12345,
				Body:     "# zgsyncの使い方\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			translation := &Translation{}
			if err := translation.FromFile(tt.filepath); err != nil {
				t.Errorf("TranslationFromFile() failed: %v", err)
			}
			if translation.Locale != tt.expected.Locale {
				t.Errorf("translation.Locale failed: got %v, want %v", translation.Locale, tt.expected.Locale)
			}
			if translation.Title != tt.expected.Title {
				t.Errorf("translation.Title failed: got %v, want %v", translation.Title, tt.expected.Title)
			}
			if translation.SourceID != tt.expected.SourceID {
				t.Errorf("translation.SourceId failed: got %v, want %v", translation.SourceID, tt.expected.SourceID)
			}
			if translation.Body != tt.expected.Body {
				t.Errorf("translation.Body failed: got %v, want %v", translation.Body, tt.expected.Body)
			}
		})
	}
}

func TestTranslationFromJson(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Translation
	}{
		{
			"testdata/translation.json",
			Translation{
				Body:   "# zgsyncの使い方\n",
				Locale: "ja",
				Title:  "zgsyncの使い方",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			translation := &Translation{}
			jsonContent, _ := os.ReadFile(tt.filepath)
			if err := translation.FromJson(string(jsonContent)); err != nil {
				t.Errorf("TranslationFromJson() failed: %v", err)
			}
			if translation.Locale != tt.expected.Locale {
				t.Errorf("translation.Locale failed: got %v, want %v", translation.Locale, tt.expected.Locale)
			}
			if translation.Title != tt.expected.Title {
				t.Errorf("translation.Title failed: got %v, want %v", translation.Title, tt.expected.Title)
			}
			if translation.Body != tt.expected.Body {
				t.Errorf("translation.Body failed: got %v, want %v", translation.Body, tt.expected.Body)
			}
		})
	}
}

func TestTranslationFromFile_ErrorCases(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name        string
		filepath    string
		expectError bool
	}{
		{
			name:        "non-existent file",
			filepath:    "testdata/non-existent.md",
			expectError: true,
		},
		{
			name:        "invalid frontmatter",
			filepath:    "testdata/invalid-frontmatter.md",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			translation := &Translation{}
			err := translation.FromFile(tt.filepath)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestTranslationFromJson_ErrorCases(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name        string
		jsonContent string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			jsonContent: `{"translation": invalid json`,
			expectError: true,
		},
		{
			name:        "missing translation wrapper",
			jsonContent: `{"title": "Test"}`,
			expectError: false, // This should still work
		},
		{
			name:        "empty JSON",
			jsonContent: `{}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			translation := &Translation{}
			err := translation.FromJson(tt.jsonContent)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestTranslationToPayload(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		translation    Translation
		expectContains []string
	}{
		{
			name: "basic translation",
			translation: Translation{
				Title:    "Test Translation",
				Locale:   "ja",
				SourceID: 123,
				Body:     "<h1>Test Content</h1>",
				Draft:    true,
			},
			expectContains: []string{
				`"title":"Test Translation"`,
				`"locale":"ja"`,
				`"source_id":123`,
				`"body":"\u003ch1\u003eTest Content\u003c/h1\u003e"`,
				`"draft":true`,
			},
		},
		{
			name: "translation without draft",
			translation: Translation{
				Title:    "Live Translation",
				Locale:   "en_us",
				SourceID: 456,
				Body:     "<p>Live content</p>",
				Draft:    false,
			},
			expectContains: []string{
				`"title":"Live Translation"`,
				`"locale":"en_us"`,
				`"source_id":456`,
				`"body":"\u003cp\u003eLive content\u003c/p\u003e"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			payload, err := tt.translation.ToPayload()
			if err != nil {
				t.Errorf("ToPayload() failed: %v", err)
				return
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
				t.Errorf("ToPayload() produced invalid JSON: %v", err)
			}

			// Check that expected content is present
			for _, expected := range tt.expectContains {
				if !strings.Contains(payload, expected) {
					t.Errorf("ToPayload() missing expected content: %s\nGot: %s", expected, payload)
				}
			}
		})
	}
}

func TestTranslationSave(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		translation    Translation
		appendFileName bool
		expectFileName string
	}{
		{
			name: "save with filename appended",
			translation: Translation{
				ID:       123,
				SourceID: 456,
				Title:    "Test Translation",
				Locale:   "ja",
				Body:     "# Test Content\n",
			},
			appendFileName: true,
			expectFileName: "456-ja.md",
		},
		// NOTE: appendFileName=false case is handled differently by the Save method
		// It uses the provided path as a directory and creates the file with source_id-locale format
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			tempDir := t.TempDir()
			savePath := tempDir

			err := tt.translation.Save(savePath, tt.appendFileName)
			if err != nil {
				t.Errorf("Save() failed: %v", err)
				return
			}

			// Verify file was created
			expectedPath := filepath.Join(tempDir, tt.expectFileName)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Errorf("Expected file %s was not created", expectedPath)
				return
			}

			// Verify file contents
			content, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Errorf("Failed to read saved file: %v", err)
				return
			}

			contentStr := string(content)
			
			// Check for YAML frontmatter structure
			if !strings.HasPrefix(contentStr, "---\n") {
				t.Errorf("File should start with YAML frontmatter delimiter")
			}
			
			if !strings.Contains(contentStr, "title: "+tt.translation.Title) {
				t.Errorf("File should contain title: %s", tt.translation.Title)
			}
			
			if !strings.Contains(contentStr, "locale: "+tt.translation.Locale) {
				t.Errorf("File should contain locale: %s", tt.translation.Locale)
			}
			
			if !strings.Contains(contentStr, tt.translation.Body) {
				t.Errorf("File should contain body content: %s", tt.translation.Body)
			}
		})
	}
}

func TestTranslationSave_ErrorCases(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		translation    Translation
		path           string
		appendFileName bool
		expectError    bool
	}{
		{
			name: "invalid path permissions",
			translation: Translation{
				ID:       123,
				SourceID: 456,
				Title:    "Test",
				Locale:   "ja",
			},
			path:           "/root/no-permission",  // Assuming this would fail
			appendFileName: true,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			err := tt.translation.Save(tt.path, tt.appendFileName)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
