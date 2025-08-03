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

func TestRateLimitHandling_Push(t *testing.T) {
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
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "HTTP 429 with Retry-After header",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded. Retry after 60 seconds")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "Rate limit", "Retry"},
			description:   "Should handle 429 with retry-after information",
		},
		{
			name: "HTTP 429 API quota exceeded",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: API quota exceeded for this minute")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "quota exceeded"},
			description:   "Should handle API quota exceeded errors",
		},
		{
			name: "HTTP 429 concurrent request limit",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Too many concurrent requests")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "concurrent"},
			description:   "Should handle concurrent request limit errors",
		},
		{
			name: "HTTP 429 daily limit reached",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Daily API limit reached")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "Daily", "limit"},
			description:   "Should handle daily limit errors",
		},
		{
			name: "HTTP 429 with specific rate limit response",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: You have exceeded the rate limit of 200 requests per minute")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "rate limit", "200 requests"},
			description:   "Should handle specific rate limit information",
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
					t.Logf("Rate limit error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestRateLimitHandling_Pull(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "HTTP 429 on ShowArticle with rate limit",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded. Try again in 30 seconds")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "Rate limit", "30 seconds"},
			description:   "Should handle rate limit errors when fetching articles",
		},
		{
			name: "HTTP 429 on ShowTranslation with hourly limit",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Hourly rate limit exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "Hourly", "rate limit"},
			description:   "Should handle hourly rate limit errors",
		},
		{
			name: "HTTP 429 with burst limit exceeded",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Burst limit of 10 requests per second exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "Burst limit", "10 requests"},
			description:   "Should handle burst limit errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := &CommandPull{
				Locale:     testhelper.TestLocales.Japanese,
				ArticleIDs: []int{testhelper.TestArticleID},
				client:     mockClient,
				converter:  converter.NewConverter(false),
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
					t.Logf("Rate limit error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestTimeoutHandling_Push(t *testing.T) {
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
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "Connection timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("Post request failed: context deadline exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"deadline exceeded", "timeout"},
			description:   "Should handle connection timeout errors",
		},
		{
			name: "Read timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("net/http: request canceled while waiting for connection")
				}
			},
			expectError:   true,
			errorKeywords: []string{"request canceled", "connection"},
			description:   "Should handle read timeout errors",
		},
		{
			name: "Client timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("Client.Timeout exceeded while awaiting headers")
				}
			},
			expectError:   true,
			errorKeywords: []string{"Timeout exceeded", "headers"},
			description:   "Should handle client timeout errors",
		},
		{
			name: "HTTP request timeout (408)",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 408 Request Timeout: Server timeout occurred")
				}
			},
			expectError:   true,
			errorKeywords: []string{"408", "Request Timeout"},
			description:   "Should handle HTTP 408 request timeout",
		},
		{
			name: "Gateway timeout (504)",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 504 Gateway Timeout: Upstream server timeout")
				}
			},
			expectError:   true,
			errorKeywords: []string{"504", "Gateway Timeout"},
			description:   "Should handle HTTP 504 gateway timeout",
		},
		{
			name: "DNS timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("dial tcp: lookup example.zendesk.com: i/o timeout")
				}
			},
			expectError:   true,
			errorKeywords: []string{"lookup", "i/o timeout"},
			description:   "Should handle DNS timeout errors",
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
					t.Logf("Timeout error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestTimeoutHandling_Pull(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "Show article timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("Get request failed: context deadline exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"deadline exceeded"},
			description:   "Should handle timeout when fetching articles",
		},
		{
			name: "Show translation timeout with connection reset",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return "", fmt.Errorf("read tcp: connection reset by peer")
				}
			},
			expectError:   true,
			errorKeywords: []string{"connection reset"},
			description:   "Should handle connection reset during translation fetch",
		},
		{
			name: "Slow response timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("net/http: request canceled (Client.Timeout exceeded while reading body)")
				}
			},
			expectError:   true,
			errorKeywords: []string{"request canceled", "reading body"},
			description:   "Should handle slow response timeouts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := &CommandPull{
				Locale:     testhelper.TestLocales.Japanese,
				ArticleIDs: []int{testhelper.TestArticleID},
				client:     mockClient,
				converter:  converter.NewConverter(false),
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
					t.Logf("Timeout error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestConcurrentRequestHandling(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "Too many concurrent connections",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Too many concurrent connections from your IP")
				}
			},
			expectError:   true,
			errorKeywords: []string{"429", "concurrent connections"},
			description:   "Should handle concurrent connection limits",
		},
		{
			name: "Resource temporarily unavailable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Resource temporarily unavailable due to high load")
				}
			},
			expectError:   true,
			errorKeywords: []string{"503", "temporarily unavailable", "high load"},
			description:   "Should handle temporary unavailability due to load",
		},
		{
			name: "Connection pool exhausted",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("dial tcp: connection pool exhausted")
				}
			},
			expectError:   true,
			errorKeywords: []string{"connection pool exhausted"},
			description:   "Should handle connection pool exhaustion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			cmd := &CommandPull{
				Locale:     testhelper.TestLocales.Japanese,
				ArticleIDs: []int{testhelper.TestArticleID},
				client:     mockClient,
				converter:  converter.NewConverter(false),
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
					t.Logf("Concurrent request error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}

func TestTimeoutHandling_Empty(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupMock     func(*testhelper.MockZendeskClient)
		expectError   bool
		errorKeywords []string
		description   string
	}{
		{
			name: "Create article timeout",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("Post request failed: context deadline exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"deadline exceeded"},
			description:   "Should handle timeout during article creation",
		},
		{
			name: "Show translation timeout after create",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				// First call (CreateArticle) succeeds
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return testhelper.CreateDefaultArticleResponse(123, testhelper.TestSectionID), nil
				}
				// Second call (ShowTranslation) times out
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return "", fmt.Errorf("Get request failed: context deadline exceeded")
				}
			},
			expectError:   true,
			errorKeywords: []string{"deadline exceeded"},
			description:   "Should handle timeout when fetching translation after article creation",
		},
		{
			name: "Network timeout with connection refused",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("dial tcp 127.0.0.1:443: connect: connection refused")
				}
			},
			expectError:   true,
			errorKeywords: []string{"connection refused"},
			description:   "Should handle network connectivity issues",
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
					t.Logf("Timeout error message for %s: %s", tt.name, errorMsg)
					t.Logf("Expected one of keywords: %v", tt.errorKeywords)
				}
			}
		})
	}
}
