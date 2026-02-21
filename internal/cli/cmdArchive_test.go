package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
)

func TestCommandArchive_Run(t *testing.T) {
	tempDir := t.TempDir()

	// Create test article file with ID in frontmatter
	articleFile := filepath.Join(tempDir, "123456.md")
	articleContent := `---
id: 123456
locale: en_us
permission_group_id: 789
title: Test Article
section_id: 123
---
`
	if err := os.WriteFile(articleFile, []byte(articleContent), 0644); err != nil {
		t.Fatalf("Failed to create test article file: %v", err)
	}

	// Create test article file without ID
	noIDFile := filepath.Join(tempDir, "no-id.md")
	noIDContent := `---
locale: en_us
permission_group_id: 789
title: Test Article Without ID
---
`
	if err := os.WriteFile(noIDFile, []byte(noIDContent), 0644); err != nil {
		t.Fatalf("Failed to create test article file: %v", err)
	}

	tests := []struct {
		name        string
		target      string
		expectError bool
		mockSetup   func(*testhelper.MockZendeskClient)
	}{
		{
			name:        "archive by article ID",
			target:      "123456",
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ArchiveArticleFunc = func(articleID int) error {
					if articleID != 123456 {
						t.Errorf("Expected article ID 123456, got %d", articleID)
					}
					return nil
				}
			},
		},
		{
			name:        "archive by file path",
			target:      articleFile,
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ArchiveArticleFunc = func(articleID int) error {
					if articleID != 123456 {
						t.Errorf("Expected article ID 123456, got %d", articleID)
					}
					return nil
				}
			},
		},
		{
			name:        "API error returns error",
			target:      "123456",
			expectError: true,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ArchiveArticleFunc = func(articleID int) error {
					return fmt.Errorf("unexpected status code: 404")
				}
			},
		},
		{
			name:        "file without article ID returns error",
			target:      noIDFile,
			expectError: true,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
		{
			name:        "non-existent file returns error",
			target:      "/non/existent/file.md",
			expectError: true,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)

			cmd := &CommandArchive{
				Target: tt.target,
				client: mockClient,
			}

			global := &Global{
				Config: Config{
					DefaultLocale: testhelper.TestLocales.English,
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestCommandArchive_AfterApply(t *testing.T) {
	global := &Global{
		Config: Config{
			Subdomain: "test",
			Email:     "test@example.com",
			Token:     "token",
		},
	}

	cmd := &CommandArchive{}
	err := cmd.AfterApply(global)

	if err != nil {
		t.Errorf("AfterApply() failed: %v", err)
	}
	if cmd.client == nil {
		t.Error("client should be initialized")
	}
}

func TestCommandArchive_resolveArticleID(t *testing.T) {
	tempDir := t.TempDir()

	articleFile := filepath.Join(tempDir, "article.md")
	articleContent := `---
id: 999
locale: en_us
title: Test
---
`
	if err := os.WriteFile(articleFile, []byte(articleContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		target      string
		wantID      int
		expectError bool
	}{
		{
			name:        "numeric string resolves to ID",
			target:      "42",
			wantID:      42,
			expectError: false,
		},
		{
			name:        "zero returns error",
			target:      "0",
			wantID:      0,
			expectError: true,
		},
		{
			name:        "negative number returns error",
			target:      "-1",
			wantID:      0,
			expectError: true,
		},
		{
			name:        "file path resolves to article ID from frontmatter",
			target:      articleFile,
			wantID:      999,
			expectError: false,
		},
		{
			name:        "non-existent file returns error",
			target:      "/no/such/file.md",
			wantID:      0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommandArchive{Target: tt.target}
			id, err := cmd.resolveArticleID()
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && id != tt.wantID {
				t.Errorf("Expected ID %d, got %d", tt.wantID, id)
			}
		})
	}
}
