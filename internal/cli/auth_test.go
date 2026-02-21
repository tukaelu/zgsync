package cli

import (
	"fmt"
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
				testFile := testhelper.CreateTestFile(t, tempDir, "test.md", testhelper.TestTranslationContent)

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
					Locale:     testhelper.TestLocales.Japanese,
					ArticleIDs: []int{testhelper.TestArticleID},
					client:     mock,
					converter:  converter.NewConverter(false),
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
				testFile := testhelper.CreateTestFile(t, tempDir, "test.md", testhelper.TestTranslationContent)

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

			// Validate error message contains authentication-related keywords
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				hasAuthKeyword := strings.Contains(errorMsg, "Unauthorized") ||
					strings.Contains(errorMsg, "authentication") ||
					strings.Contains(errorMsg, "credentials") ||
					strings.Contains(errorMsg, "token") ||
					strings.Contains(errorMsg, "401")

				if !hasAuthKeyword {
					t.Errorf("Authentication error message missing auth keywords: %s", errorMsg)
				}
			}
		})
	}
}
