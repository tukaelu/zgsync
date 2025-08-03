package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	"github.com/tukaelu/zgsync/internal/converter"
)

func TestCommandPush_Run(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	
	// Create test article file
	articleFile := filepath.Join(tempDir, "test-article.md")
	articleContent := `---
locale: en_us
permission_group_id: 123
title: Test Article
---
`
	if err := os.WriteFile(articleFile, []byte(articleContent), 0644); err != nil {
		t.Fatalf("Failed to create test article file: %v", err)
	}

	// Create test translation file
	translationFile := filepath.Join(tempDir, "test-translation.md")
	translationContent := `---
locale: ja
title: Test Translation
source_id: 456
---
# Test Content
This is test content.
`
	if err := os.WriteFile(translationFile, []byte(translationContent), 0644); err != nil {
		t.Fatalf("Failed to create test translation file: %v", err)
	}

	tests := []struct {
		name          string
		cmd           CommandPush
		files         []string
		expectError   bool
		mockSetup     func(*testhelper.MockZendeskClient)
	}{
		{
			name: "push article successfully",
			cmd: CommandPush{
				Article: true,
				DryRun:  false,
				Raw:     false,
			},
			files:       []string{articleFile},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateArticleFunc = func(locale string, articleID int, payload string) (string, error) {
					return testhelper.CreateDefaultArticleResponse(123, 456), nil
				}
			},
		},
		{
			name: "push translation successfully",
			cmd: CommandPush{
				Article: false,
				DryRun:  false,
				Raw:     false,
			},
			files:       []string{translationFile},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
		},
		{
			name: "dry run mode",
			cmd: CommandPush{
				Article: false,
				DryRun:  true,
				Raw:     false,
			},
			files:       []string{translationFile},
			expectError: false,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
		{
			name: "raw mode translation",
			cmd: CommandPush{
				Article: false,
				DryRun:  false,
				Raw:     true,
			},
			files:       []string{translationFile},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
		},
		{
			name: "non-existent file",
			cmd: CommandPush{
				Article: false,
				DryRun:  false,
				Raw:     false,
			},
			files:       []string{"/non/existent/file.md"},
			expectError: true,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
		{
			name: "directory path instead of file",
			cmd: CommandPush{
				Article: false,
				DryRun:  false,
				Raw:     false,
			},
			files:       []string{tempDir}, // Pass directory instead of file
			expectError: true,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)
			
			cmd := tt.cmd
			cmd.Files = tt.files
			cmd.client = mockClient
			cmd.converter = converter.NewConverter(false)

			global := &Global{
				Config: Config{
					DefaultLocale:     testhelper.TestLocales.English,
					NotifySubscribers: false,
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

func TestCommandPush_AfterApply(t *testing.T) {
	global := &Global{
		Config: Config{
			Subdomain:             "test",
			Email:                 "test@example.com",
			Token:                 "token",
			EnableLinkTargetBlank: true,
		},
	}

	cmd := &CommandPush{}
	err := cmd.AfterApply(global)
	
	if err != nil {
		t.Errorf("AfterApply() failed: %v", err)
	}
	
	if cmd.client == nil {
		t.Error("client should be initialized")
	}
	
	if cmd.converter == nil {
		t.Error("converter should be initialized")
	}
}

func TestCommandPush_FilePermissionErrors(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a file with restricted read permissions
	restrictedFile := filepath.Join(tempDir, "restricted.md")
	restrictedContent := `---
locale: ja
title: Restricted File
source_id: 789
---
# Restricted Content
`
	if err := os.WriteFile(restrictedFile, []byte(restrictedContent), 0644); err != nil {
		t.Fatalf("Failed to create restricted file: %v", err)
	}
	
	// Remove read permissions (this test may not work on all platforms)
	if err := os.Chmod(restrictedFile, 0000); err != nil {
		t.Skipf("Cannot change file permissions on this platform: %v", err)
	}
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(restrictedFile, 0644)
	}()

	cmd := CommandPush{
		Article: false,
		DryRun:  false,
		Raw:     false,
		Files:   []string{restrictedFile},
	}
	cmd.client = &testhelper.MockZendeskClient{}
	cmd.converter = converter.NewConverter(false)

	global := &Global{
		Config: Config{
			DefaultLocale:     testhelper.TestLocales.English,
			NotifySubscribers: false,
		},
	}

	err := cmd.Run(global)
	if err == nil {
		t.Error("Expected permission error but got none")
	}
}

func TestCommandPush_pushArticle(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test article file with ID
	articleFile := filepath.Join(tempDir, "test-article.md")
	articleContent := `---
id: 123
locale: en_us
permission_group_id: 456
title: Test Article
---
`
	if err := os.WriteFile(articleFile, []byte(articleContent), 0644); err != nil {
		t.Fatalf("Failed to create test article file: %v", err)
	}

	tests := []struct {
		name        string
		dryRun      bool
		expectError bool
		mockSetup   func(*testhelper.MockZendeskClient)
	}{
		{
			name:        "successful article push",
			dryRun:      false,
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateArticleFunc = func(locale string, articleID int, payload string) (string, error) {
					if articleID != 123 {
						t.Errorf("Expected article ID 123, got %d", articleID)
					}
					return testhelper.CreateDefaultArticleResponse(123, 456), nil
				}
			},
		},
		{
			name:        "dry run mode",
			dryRun:      true,
			expectError: false,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)
			
			cmd := &CommandPush{
				DryRun: tt.dryRun,
				client: mockClient,
			}

			global := &Global{
				Config: Config{
					DefaultLocale:     testhelper.TestLocales.English,
					NotifySubscribers: false,
				},
			}

			err := cmd.pushArticle(global, articleFile)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestCommandPush_pushTranslation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test translation file
	translationFile := filepath.Join(tempDir, "test-translation.md")
	translationContent := `---
locale: ja
title: Test Translation
source_id: 456
---
# Test Content
This is test content.
`
	if err := os.WriteFile(translationFile, []byte(translationContent), 0644); err != nil {
		t.Fatalf("Failed to create test translation file: %v", err)
	}

	tests := []struct {
		name        string
		dryRun      bool
		raw         bool
		expectError bool
		mockSetup   func(*testhelper.MockZendeskClient)
	}{
		{
			name:        "successful translation push",
			dryRun:      false,
			raw:         false,
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					if articleID != 456 {
						t.Errorf("Expected source ID 456, got %d", articleID)
					}
					if locale != testhelper.TestLocales.Japanese {
						t.Errorf("Expected locale '%s', got '%s'", testhelper.TestLocales.Japanese, locale)
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
		},
		{
			name:        "raw mode translation push",
			dryRun:      false,
			raw:         true,
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
		},
		{
			name:        "dry run mode",
			dryRun:      true,
			raw:         false,
			expectError: false,
			mockSetup:   func(mock *testhelper.MockZendeskClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)
			
			cmd := &CommandPush{
				DryRun:    tt.dryRun,
				Raw:       tt.raw,
				client:    mockClient,
				converter: converter.NewConverter(false),
			}

			global := &Global{
				Config: Config{
					DefaultLocale:     testhelper.TestLocales.English,
					NotifySubscribers: false,
				},
			}

			err := cmd.pushTranslation(global, translationFile)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}