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

func TestAPIResponseErrors_Push(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test translation file
	testFile := filepath.Join(tempDir, "test.md")
	testContent := `---
locale: ja
title: "Test Translation"
source_id: 123
---
# Test Content`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		errorKeywords []string
		description string
	}{
		{
			name: "HTTP 500 Internal Server Error",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 500 Internal Server Error: unexpected status code: 500")
				}
			},
			expectError: true,
			errorKeywords: []string{"500", "Internal Server Error"},
			description: "Should handle 500 server errors gracefully",
		},
		{
			name: "HTTP 404 Article Not Found",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 404 Not Found: Article with ID %d not found", articleID)
				}
			},
			expectError: true,
			errorKeywords: []string{"404", "Not Found"},
			description: "Should handle 404 resource not found errors",
		},
		{
			name: "HTTP 503 Service Unavailable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Service temporarily unavailable")
				}
			},
			expectError: true,
			errorKeywords: []string{"503", "Service Unavailable"},
			description: "Should handle 503 service unavailable errors",
		},
		{
			name: "HTTP 429 Rate Limited",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded")
				}
			},
			expectError: true,
			errorKeywords: []string{"429", "Too Many Requests", "Rate limit"},
			description: "Should handle 429 rate limit errors",
		},
		{
			name: "Network connection timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("network error: dial tcp: i/o timeout")
				}
			},
			expectError: true,
			errorKeywords: []string{"network", "timeout"},
			description: "Should handle network timeout errors",
		},
		{
			name: "Connection refused error",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("network error: dial tcp 127.0.0.1:443: connect: connection refused")
				}
			},
			expectError: true,
			errorKeywords: []string{"connection refused"},
			description: "Should handle connection refused errors",
		},
		{
			name: "Malformed JSON response",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					// Return malformed JSON that might cause parsing issues downstream
					return "invalid json response", nil
				}
			},
			expectError: false, // The command might succeed but with invalid JSON
			description: "Should handle malformed JSON responses",
		},
		{
			name: "Empty response body",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", nil // Empty response
				}
			},
			expectError: false, // Empty response might be valid for some operations
			description: "Should handle empty response bodies",
		},
		{
			name: "HTTP 400 Bad Request with details",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 400 Bad Request: Invalid article data - missing required field 'title'")
				}
			},
			expectError: true,
			errorKeywords: []string{"400", "Bad Request", "Invalid"},
			description: "Should handle 400 bad request errors with detailed messages",
		},
		{
			name: "HTTP 422 Unprocessable Entity",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 422 Unprocessable Entity: Validation failed - title cannot be blank")
				}
			},
			expectError: true,
			errorKeywords: []string{"422", "Unprocessable Entity", "Validation"},
			description: "Should handle 422 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := CommandPush{
				Article: false,
				DryRun:  false,
				Raw:     false,
				Files:   []string{testFile},
			}
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
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			// Additional validation: check that error message contains expected keywords
			if tt.expectError && err != nil && len(tt.errorKeywords) > 0 {
				errorMsg := err.Error()
				foundKeyword := false
				for _, keyword := range tt.errorKeywords {
					if strings.Contains(errorMsg, keyword) {
						foundKeyword = true
						break
					}
				}
				if !foundKeyword {
					t.Logf("API error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestAPIResponseErrors_Pull(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		errorKeywords []string
		description string
	}{
		{
			name: "HTTP 500 on ShowArticle",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 500 Internal Server Error: Database connection failed")
				}
			},
			expectError: true,
			errorKeywords: []string{"500", "Internal Server Error"},
			description: "Should handle 500 errors when fetching articles",
		},
		{
			name: "HTTP 404 on ShowTranslation", 
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return "", fmt.Errorf("HTTP 404 Not Found: Translation not found for article %d in locale %s", articleID, locale)
				}
			},
			expectError: true,
			errorKeywords: []string{"404", "Not Found"},
			description: "Should handle 404 errors when translation doesn't exist",
		},
		{
			name: "Network timeout on ShowArticle",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("Get request failed: context deadline exceeded")
				}
			},
			expectError: true,
			errorKeywords: []string{"deadline exceeded", "timeout"},
			description: "Should handle network timeouts",
		},
		{
			name: "Malformed JSON response from ShowArticle",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "{invalid json structure", nil // Malformed JSON
				}
			},
			expectError: true, // This will likely cause parsing errors in FromJson
			description: "Should handle malformed JSON from API",
		},
		{
			name: "Empty response from ShowTranslation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return "", nil // Empty response
				}
			},
			expectError: true, // Empty JSON will likely cause parsing errors
			description: "Should handle empty responses from API",
		},
		{
			name: "HTTP 403 Forbidden Access",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 403 Forbidden: Access denied - insufficient permissions")
				}
			},
			expectError: true,
			errorKeywords: []string{"403", "Forbidden", "Access denied"},
			description: "Should handle 403 permission errors",
		},
		{
			name: "HTTP 502 Bad Gateway",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 502 Bad Gateway: Upstream server error")
				}
			},
			expectError: true,
			errorKeywords: []string{"502", "Bad Gateway"},
			description: "Should handle 502 gateway errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := &CommandPull{
				Locale:      testhelper.TestLocales.Japanese,
				ArticleIDs:  []int{testhelper.TestArticleID},
				client:      mockClient,
				converter:   converter.NewConverter(false),
			}

			global := &Global{
				Config: Config{
					DefaultLocale: testhelper.TestLocales.English,
					ContentsDir:   tempDir,
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			// Additional validation: check that error message contains expected keywords
			if tt.expectError && err != nil && len(tt.errorKeywords) > 0 {
				errorMsg := err.Error()
				foundKeyword := false
				for _, keyword := range tt.errorKeywords {
					if strings.Contains(errorMsg, keyword) {
						foundKeyword = true
						break
					}
				}
				if !foundKeyword {
					t.Logf("API error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestAPIResponseErrors_Empty(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		errorKeywords []string
		description string
	}{
		{
			name: "HTTP 500 on CreateArticle",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 500 Internal Server Error: Article creation failed")
				}
			},
			expectError: true,
			errorKeywords: []string{"500", "Internal Server Error"},
			description: "Should handle 500 errors during article creation",
		},
		{
			name: "HTTP 404 Section Not Found",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 404 Not Found: Section with ID %d not found", sectionID)
				}
			},
			expectError: true,
			errorKeywords: []string{"404", "Not Found", "Section"},
			description: "Should handle 404 errors when section doesn't exist",
		},
		{
			name: "HTTP 403 Forbidden Create Permission",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 403 Forbidden: User does not have permission to create articles in this section")
				}
			},
			expectError: true,
			errorKeywords: []string{"403", "Forbidden", "permission"},
			description: "Should handle 403 permission errors for article creation",
		},
		{
			name: "Network connection error",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("network error: no route to host")
				}
			},
			expectError: true,
			errorKeywords: []string{"network", "no route to host"},
			description: "Should handle network connectivity errors",
		},
		{
			name: "Invalid JSON response from CreateArticle",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "invalid json response body", nil // Malformed response
				}
			},
			expectError: true, // The command will fail when trying to parse JSON in a.FromJson(res)
			errorKeywords: []string{"invalid character"},
			description: "Should handle invalid JSON responses",
		},
		{
			name: "HTTP 422 Validation Error",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 422 Unprocessable Entity: Title cannot be blank")
				}
			},
			expectError: true,
			errorKeywords: []string{"422", "Unprocessable Entity", "Title"},
			description: "Should handle 422 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := &CommandEmpty{
				SectionID: testhelper.TestSectionID,
				Title:     "Test Article",
				Locale:    testhelper.TestLocales.English,
				client:    mockClient,
			}

			global := &Global{
				Config: Config{
					DefaultLocale:            testhelper.TestLocales.English,
					DefaultPermissionGroupID: 123,
					ContentsDir:              tempDir,
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			// Additional validation: check that error message contains expected keywords
			if tt.expectError && err != nil && len(tt.errorKeywords) > 0 {
				errorMsg := err.Error()
				foundKeyword := false
				for _, keyword := range tt.errorKeywords {
					if strings.Contains(errorMsg, keyword) {
						foundKeyword = true
						break
					}
				}
				if !foundKeyword {
					t.Logf("API error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestAPIResponseErrors_JSONParsing(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name        string
		command     string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "Pull command with corrupted JSON response",
			command: "pull",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Return JSON with syntax errors
					return `{"article": {"id": 123, "title": "Unclosed quote}`, nil
				}
			},
			expectError: true,
			description: "Should handle JSON parsing errors in pull command",
		},
		{
			name: "Pull command with unexpected JSON structure",
			command: "pull",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Return valid JSON but with unexpected structure
					return `{"unexpected_field": {"not_article": "data"}}`, nil
				}
			},
			expectError: false, // Article.FromJson doesn't fail with unexpected structure
			description: "Should handle unexpected JSON structure gracefully",
		},
		{
			name: "Pull command with null response fields",
			command: "pull",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Return JSON with null values in critical fields
					return `{"article": {"id": null, "title": null, "locale": null}}`, nil
				}
			},
			expectError: false, // Article.FromJson handles null fields without error
			description: "Should handle null values in critical JSON fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			switch tt.command {
			case "pull":
				cmd := &CommandPull{
					Locale:      testhelper.TestLocales.Japanese,
					ArticleIDs:  []int{testhelper.TestArticleID},
					client:      mockClient,
					converter:   converter.NewConverter(false),
				}

				global := &Global{
					Config: Config{
						DefaultLocale: testhelper.TestLocales.English,
						ContentsDir:   tempDir,
					},
				}

				err := cmd.Run(global)
				if tt.expectError && err == nil {
					t.Errorf("Expected error for %s but got none", tt.name)
				}
				if !tt.expectError && err != nil {
					t.Errorf("Expected no error for %s but got: %v", tt.name, err)
				}
			}
		})
	}
}