package zendesk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compareUserSegmentID safely compares two potentially nil int pointers
func compareUserSegmentID(got, want *int) bool {
	if got == nil && want == nil {
		return true
	}
	if got == nil || want == nil {
		return false
	}
	return *got == *want
}

func TestArticleFromFile(t *testing.T) {
	refUserSegmentID := 123
	tests := []struct {
		filepath string
		expected Article
	}{
		{
			"testdata/article-ja.md",
			Article{
				Locale:            "ja",
				PermissionGroupID: 12345,
				Title:             "zgsyncの使い方",
				UserSegmentIDs:    []int{123, 456},
			},
		},
		{
			"testdata/article-en.md",
			Article{
				Locale:            "en_us",
				PermissionGroupID: 56,
				Title:             "How to use zgsync",
				UserSegmentID:     &refUserSegmentID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			article := &Article{}
			if err := article.FromFile(tt.filepath); err != nil {
				t.Errorf("ArticleFromFile() failed: %v", err)
			}
			if article.Locale != tt.expected.Locale {
				t.Errorf("article.Locale failed: got %v, want %v", article.Locale, tt.expected.Locale)
			}
			if article.PermissionGroupID != tt.expected.PermissionGroupID {
				t.Errorf("article.PermissionGroupId failed: got %v, want %v", article.PermissionGroupID, tt.expected.PermissionGroupID)
			}
			if article.Title != tt.expected.Title {
				t.Errorf("article.Title failed: got %v, want %v", article.Title, tt.expected.Title)
			}
			if !compareUserSegmentID(article.UserSegmentID, tt.expected.UserSegmentID) {
				t.Errorf("article.UserSegmentId failed: got %v, want %v", article.UserSegmentID, tt.expected.UserSegmentID)
			}
			if len(article.UserSegmentIDs) != len(tt.expected.UserSegmentIDs) {
				t.Errorf("article.UserSegmentIds failed: got %v, want %v", article.UserSegmentIDs, tt.expected.UserSegmentIDs)
			}
		})
	}
}

func TestArticleFromJson(t *testing.T) {
	refUserSegmentID := 12
	tests := []struct {
		filepath string
		expected Article
	}{
		{
			"testdata/article.json",
			Article{
				AuthorID:          3465,
				CommentsDisabled:  true,
				ContentTagIDs:     []string{"01GT23D51Y", "01GT23FWWN"},
				ID:                37486578,
				Locale:            "en_us",
				PermissionGroupID: 123,
				Position:          42,
				Promoted:          false,
				Title:             "How to use zgsync",
				UserSegmentID:     &refUserSegmentID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			article := &Article{}
			jsonContent, _ := os.ReadFile(tt.filepath)
			if err := article.FromJson(string(jsonContent)); err != nil {
				t.Errorf("ArticleFromJson() failed: %v", err)
			}
			if article.AuthorID != tt.expected.AuthorID {
				t.Errorf("article.AuthorID failed: got %v, want %v", article.AuthorID, tt.expected.AuthorID)
			}
			if article.CommentsDisabled != tt.expected.CommentsDisabled {
				t.Errorf("article.CommentsDisabled failed: got %v, want %v", article.CommentsDisabled, tt.expected.CommentsDisabled)
			}
			if len(article.ContentTagIDs) != len(tt.expected.ContentTagIDs) {
				t.Errorf("article.ContentTagIds failed: got %v, want %v", article.ContentTagIDs, tt.expected.ContentTagIDs)
			}
			if article.ID != tt.expected.ID {
				t.Errorf("article.ID failed: got %v, want %v", article.ID, tt.expected.ID)
			}
			if article.Locale != tt.expected.Locale {
				t.Errorf("article.Locale failed: got %v, want %v", article.Locale, tt.expected.Locale)
			}
			if article.PermissionGroupID != tt.expected.PermissionGroupID {
				t.Errorf("article.PermissionGroupId failed: got %v, want %v", article.PermissionGroupID, tt.expected.PermissionGroupID)
			}
			if article.Position != tt.expected.Position {
				t.Errorf("article.Position failed: got %v, want %v", article.Position, tt.expected.Position)
			}
			if article.Promoted != tt.expected.Promoted {
				t.Errorf("article.Promoted failed: got %v, want %v", article.Promoted, tt.expected.Promoted)
			}
			if article.Title != tt.expected.Title {
				t.Errorf("article.Title failed: got %v, want %v", article.Title, tt.expected.Title)
			}
			if !compareUserSegmentID(article.UserSegmentID, tt.expected.UserSegmentID) {
				t.Errorf("article.UserSegmentId failed: got %v, want %v", article.UserSegmentID, tt.expected.UserSegmentID)
			}
		})
	}
}

func TestArticleFromFile_ErrorCases(t *testing.T) {
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
			
			article := &Article{}
			err := article.FromFile(tt.filepath)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestArticleFromJson_ErrorCases(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name        string
		jsonContent string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			jsonContent: `{"article": invalid json`,
			expectError: true,
		},
		{
			name:        "missing article wrapper",
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
			
			article := &Article{}
			err := article.FromJson(tt.jsonContent)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestArticleToPayload(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		article        Article
		notify         bool
		expectContains []string
	}{
		{
			name: "basic article with notification",
			article: Article{
				ID:                123,
				Title:             "Test Article",
				Locale:            "en_us",
				PermissionGroupID: 456,
				Draft:             true,
			},
			notify: true,
			expectContains: []string{
				`"id":123`,
				`"title":"Test Article"`,
				`"locale":"en_us"`,
				`"permission_group_id":456`,
				`"draft":true`,
				`"notify_subscribers":true`,
			},
		},
		{
			name: "article without notification",
			article: Article{
				ID:     789,
				Title:  "No Notify",
				Locale: "ja",
			},
			notify: false,
			expectContains: []string{
				`"id":789`,
				`"title":"No Notify"`,
				`"locale":"ja"`,
				// Note: notify_subscribers is omitted when false due to omitempty tag
			},
		},
		{
			name: "article with user segment",
			article: Article{
				Title:         "Segment Test",
				Locale:        "fr",
				UserSegmentID: func() *int { i := 999; return &i }(),
			},
			notify: false,
			expectContains: []string{
				`"title":"Segment Test"`,
				`"locale":"fr"`,
				`"user_segment_id":999`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			payload, err := tt.article.ToPayload(tt.notify)
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

func TestArticleSave(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		article        Article
		appendFileName bool
		expectFileName string
	}{
		{
			name: "save with filename appended",
			article: Article{
				ID:                123,
				Title:             "Test Article",
				Locale:            "en_us",
				PermissionGroupID: 456,
				Draft:             true,
			},
			appendFileName: true,
			expectFileName: "123.md",
		},
		// NOTE: appendFileName=false case is handled differently by the Save method
		// It uses the provided path as a directory and creates the file with ID as name
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			tempDir := t.TempDir()
			savePath := tempDir

			err := tt.article.Save(savePath, tt.appendFileName)
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
			
			if !strings.Contains(contentStr, "title: "+tt.article.Title) {
				t.Errorf("File should contain title: %s", tt.article.Title)
			}
			
			if !strings.Contains(contentStr, "locale: "+tt.article.Locale) {
				t.Errorf("File should contain locale: %s", tt.article.Locale)
			}
		})
	}
}

func TestArticleSave_ErrorCases(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		article        Article
		path           string
		appendFileName bool
		expectError    bool
	}{
		{
			name: "invalid path permissions",
			article: Article{
				ID:     123,
				Title:  "Test",
				Locale: "en_us",
			},
			path:           "/root/no-permission",  // Assuming this would fail
			appendFileName: true,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			err := tt.article.Save(tt.path, tt.appendFileName)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
