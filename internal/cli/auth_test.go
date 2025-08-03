package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	"github.com/tukaelu/zgsync/internal/converter"
)

func TestAuthentication_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		command     func(*testhelper.MockZendeskClient) error
		expectError bool
		description string
	}{
		{
			name: "push command with 401 unauthorized",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 401 Unauthorized: Authentication credentials invalid")
				}
			},
			command: func(mock *testhelper.MockZendeskClient) error {
				tempDir := t.TempDir()
				testFile := createTestFile(t, tempDir, "test.md", `---
locale: ja
title: "Test Translation"
source_id: 123
---
# Test Content`)

				cmd := CommandPush{
					Article: false,
					DryRun:  false,
					Raw:     false,
					Files:   []string{testFile},
				}
				cmd.client = mock
				cmd.converter = converter.NewConverter(false)

				global := &Global{
					Config: Config{
						DefaultLocale:     testhelper.TestLocales.English,
						NotifySubscribers: false,
					},
				}

				return cmd.Run(global)
			},
			expectError: true,
			description: "Should handle 401 authentication errors in push command",
		},
		{
			name: "pull command with 401 unauthorized",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 401 Unauthorized: Invalid API token")
				}
			},
			command: func(mock *testhelper.MockZendeskClient) error {
				tempDir := t.TempDir()

				cmd := &CommandPull{
					Locale:      testhelper.TestLocales.Japanese,
					ArticleIDs:  []int{testhelper.TestArticleID},
					client:      mock,
					converter:   converter.NewConverter(false),
				}

				global := &Global{
					Config: Config{
						DefaultLocale: testhelper.TestLocales.English,
						ContentsDir:   tempDir,
					},
				}

				return cmd.Run(global)
			},
			expectError: true,
			description: "Should handle 401 authentication errors in pull command",
		},
		{
			name: "empty command with 401 unauthorized",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 401 Unauthorized: Access denied")
				}
			},
			command: func(mock *testhelper.MockZendeskClient) error {
				tempDir := t.TempDir()

				cmd := &CommandEmpty{
					SectionID: testhelper.TestSectionID,
					client:    mock,
				}

				global := &Global{
					Config: Config{
						DefaultLocale:            testhelper.TestLocales.English,
						DefaultPermissionGroupID: 123,
						ContentsDir:              tempDir,
					},
				}

				return cmd.Run(global)
			},
			expectError: true,
			description: "Should handle 401 authentication errors in empty command",
		},
		{
			name: "push command with invalid token format error",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 401 Unauthorized: Malformed authorization header")
				}
			},
			command: func(mock *testhelper.MockZendeskClient) error {
				tempDir := t.TempDir()
				testFile := createTestFile(t, tempDir, "test.md", `---
locale: ja
title: "Test Translation"
source_id: 123
---
# Test Content`)

				cmd := CommandPush{
					Article: false,
					DryRun:  false,
					Raw:     false,
					Files:   []string{testFile},
				}
				cmd.client = mock
				cmd.converter = converter.NewConverter(false)

				global := &Global{
					Config: Config{
						DefaultLocale:     testhelper.TestLocales.English,
						NotifySubscribers: false,
					},
				}

				return cmd.Run(global)
			},
			expectError: true,
			description: "Should handle malformed authorization header errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			err := tt.command(mockClient)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			// Additional validation: check that error message contains authentication-related keywords
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				hasAuthKeyword := strings.Contains(errorMsg, "Unauthorized") || 
								strings.Contains(errorMsg, "authentication") || 
								strings.Contains(errorMsg, "credentials") ||
								strings.Contains(errorMsg, "token") ||
								strings.Contains(errorMsg, "401")
				
				if !hasAuthKeyword {
					t.Logf("Authentication error message: %s", errorMsg)
				}
			}
		})
	}
}

func TestAuthentication_TokenValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		description string
	}{
		{
			name: "empty token in config",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com/token",
				Token:                    "", // Empty token
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectError: true,
			description: "Should fail validation with empty token",
		},
		{
			name: "valid token in config",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com/token",
				Token:                    "valid_token_123",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectError: false,
			description: "Should pass validation with valid token",
		},
		{
			name: "missing email token suffix",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com", // Missing /token suffix
				Token:                    "valid_token_123",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectError: false, // Email format validation is not currently enforced
			description: "Email without /token suffix (currently allowed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validation()

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}
		})
	}
}

// Helper function to create test files
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}