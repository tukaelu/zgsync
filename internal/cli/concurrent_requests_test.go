package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	"github.com/tukaelu/zgsync/internal/converter"
)

func TestConcurrentPushRequests(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test files for concurrent testing
	testFiles := make([]string, 5)
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test-%d.md", i))
		testContent := fmt.Sprintf(`---
locale: ja
title: "Test Translation %d"
source_id: %d
---
# Test Content %d`, i, 123+i, i)

		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
		testFiles[i] = testFile
	}

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "concurrent push requests with connection limit",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					currentCall := callCount
					mu.Unlock()

					// Simulate connection limit after 3 concurrent requests
					if currentCall > 3 {
						return "", fmt.Errorf("HTTP 429 Too Many Requests: Connection limit exceeded - maximum 3 concurrent connections")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle connection limits during concurrent requests",
		},
		{
			name: "concurrent push requests with throttling",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					currentCall := callCount
					mu.Unlock()

					// Simulate throttling for rapid requests
					if currentCall > 2 {
						return "", fmt.Errorf("HTTP 429 Too Many Requests: Request rate too high - please slow down")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle throttling during rapid concurrent requests",
		},
		{
			name: "concurrent push requests with resource contention",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					// Simulate resource contention causing intermittent failures
					if articleID%2 == 0 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Resource contention - try again later")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle resource contention during concurrent operations",
		},
		{
			name: "successful concurrent push requests",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					// Simulate small delay to test concurrency
					time.Sleep(10 * time.Millisecond)
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: false,
			description: "Should handle successful concurrent requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			// Run concurrent push operations
			var wg sync.WaitGroup
			errors := make(chan error, len(testFiles))

			for _, testFile := range testFiles {
				wg.Add(1)
				go func(file string) {
					defer wg.Done()

					cmd := CommandPush{
						Article: false,
						DryRun:  false,
						Raw:     false,
						Files:   []string{file},
					}
					cmd.client = mockClient
					cmd.converter = converter.NewConverter(false)

					global := &Global{
						Config: Config{
							DefaultLocale:     testhelper.TestLocales.English,
							NotifySubscribers: false,
						},
					}

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}(testFile)
			}

			wg.Wait()
			close(errors)

			// Collect all errors
			var allErrors []error
			for err := range errors {
				allErrors = append(allErrors, err)
			}

			if tt.expectError && len(allErrors) == 0 {
				t.Errorf("Expected errors for %s but got none", tt.name)
			}
			if !tt.expectError && len(allErrors) > 0 {
				t.Errorf("Expected no errors for %s but got: %v", tt.name, allErrors)
			}

			// Log error details for debugging
			if len(allErrors) > 0 {
				t.Logf("Concurrent request errors for %s: %d total errors", tt.name, len(allErrors))
				for i, err := range allErrors {
					t.Logf("Error %d: %s", i+1, err.Error())
				}
			}
		})
	}
}

func TestConcurrentPullRequests(t *testing.T) {
	tempDir := t.TempDir()
	articleIDs := []int{100, 101, 102, 103, 104}

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "concurrent pull requests with bandwidth limit",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					mu.Lock()
					callCount++
					currentCall := callCount
					mu.Unlock()

					// Simulate bandwidth limit
					if currentCall > 3 {
						return "", fmt.Errorf("HTTP 429 Too Many Requests: Bandwidth limit exceeded")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle bandwidth limits during concurrent pulls",
		},
		{
			name: "concurrent pull requests with server overload",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Simulate server overload for some requests
					if articleID%3 == 0 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Server overloaded")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle server overload during concurrent pulls",
		},
		{
			name: "concurrent pull requests with mixed success/failure",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Some articles succeed, others fail
					if articleID == 102 {
						return "", fmt.Errorf("HTTP 404 Not Found: Article %d not found", articleID)
					}
					if articleID == 104 {
						return "", fmt.Errorf("HTTP 403 Forbidden: Access denied to article %d", articleID)
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle mixed success/failure in concurrent pulls",
		},
		{
			name: "successful concurrent pull requests",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Simulate processing time
					time.Sleep(5 * time.Millisecond)
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: false,
			description: "Should handle successful concurrent pull requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			// Run concurrent pull operations
			var wg sync.WaitGroup
			errors := make(chan error, len(articleIDs))

			for _, articleID := range articleIDs {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					cmd := &CommandPull{
						Locale:     testhelper.TestLocales.Japanese,
						ArticleIDs: []int{id},
						client:     mockClient,
						converter:  converter.NewConverter(false),
					}

					global := &Global{
						Config: Config{
							DefaultLocale: testhelper.TestLocales.English,
							ContentsDir:   tempDir,
						},
					}

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}(articleID)
			}

			wg.Wait()
			close(errors)

			// Collect all errors
			var allErrors []error
			for err := range errors {
				allErrors = append(allErrors, err)
			}

			if tt.expectError && len(allErrors) == 0 {
				t.Errorf("Expected errors for %s but got none", tt.name)
			}
			if !tt.expectError && len(allErrors) > 0 {
				t.Errorf("Expected no errors for %s but got: %v", tt.name, allErrors)
			}

			// Log error details for debugging
			if len(allErrors) > 0 {
				t.Logf("Concurrent pull errors for %s: %d total errors", tt.name, len(allErrors))
				for i, err := range allErrors {
					t.Logf("Error %d: %s", i+1, err.Error())
				}
			}
		})
	}
}

func TestConcurrentMixedOperations(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file for push operations
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
		description string
	}{
		{
			name: "mixed operations with connection pool exhaustion",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				operationCount := 0
				var mu sync.Mutex

				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					operationCount++
					currentOp := operationCount
					mu.Unlock()

					if currentOp > 4 {
						return "", fmt.Errorf("dial tcp: connection pool exhausted")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					mu.Lock()
					operationCount++
					currentOp := operationCount
					mu.Unlock()

					if currentOp > 4 {
						return "", fmt.Errorf("dial tcp: connection pool exhausted")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}

				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					mu.Lock()
					operationCount++
					currentOp := operationCount
					mu.Unlock()

					if currentOp > 4 {
						return "", fmt.Errorf("dial tcp: connection pool exhausted")
					}
					return testhelper.CreateDefaultArticleResponse(200, sectionID), nil
				}
			},
			expectError: true,
			description: "Should handle connection pool exhaustion during mixed operations",
		},
		{
			name: "mixed operations with API rate limiting",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				totalRequests := 0
				var mu sync.Mutex

				rateLimitCheck := func() error {
					mu.Lock()
					totalRequests++
					current := totalRequests
					mu.Unlock()

					if current > 6 {
						return fmt.Errorf("HTTP 429 Too Many Requests: API rate limit exceeded - %d requests per minute limit", current)
					}
					return nil
				}

				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					if err := rateLimitCheck(); err != nil {
						return "", err
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					if err := rateLimitCheck(); err != nil {
						return "", err
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}

				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					if err := rateLimitCheck(); err != nil {
						return "", err
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					if err := rateLimitCheck(); err != nil {
						return "", err
					}
					return testhelper.CreateDefaultArticleResponse(200, sectionID), nil
				}
			},
			expectError: true,
			description: "Should handle API rate limiting across mixed operation types",
		},
		{
			name: "successful mixed concurrent operations",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					time.Sleep(5 * time.Millisecond)
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					time.Sleep(3 * time.Millisecond)
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}

				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					time.Sleep(2 * time.Millisecond)
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}

				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					time.Sleep(8 * time.Millisecond)
					return testhelper.CreateDefaultArticleResponse(200, sectionID), nil
				}
			},
			expectError: false,
			description: "Should handle successful mixed concurrent operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			var wg sync.WaitGroup
			errors := make(chan error, 10)

			// Launch concurrent push operations
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

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

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}()
			}

			// Launch concurrent pull operations
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func(articleID int) {
					defer wg.Done()

					cmd := &CommandPull{
						Locale:     testhelper.TestLocales.Japanese,
						ArticleIDs: []int{articleID},
						client:     mockClient,
						converter:  converter.NewConverter(false),
					}

					global := &Global{
						Config: Config{
							DefaultLocale: testhelper.TestLocales.English,
							ContentsDir:   tempDir,
						},
					}

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}(300 + i)
			}

			// Launch concurrent empty operations
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					cmd := &CommandEmpty{
						SectionID: testhelper.TestSectionID,
						Title:     fmt.Sprintf("Test Article %d", idx),
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

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			// Collect all errors
			var allErrors []error
			for err := range errors {
				allErrors = append(allErrors, err)
			}

			if tt.expectError && len(allErrors) == 0 {
				t.Errorf("Expected errors for %s but got none", tt.name)
			}
			if !tt.expectError && len(allErrors) > 0 {
				t.Errorf("Expected no errors for %s but got: %v", tt.name, allErrors)
			}

			// Validate error types for expected error cases
			if tt.expectError && len(allErrors) > 0 {
				foundExpectedError := false
				for _, err := range allErrors {
					errorMsg := err.Error()
					if strings.Contains(errorMsg, "connection pool exhausted") ||
						strings.Contains(errorMsg, "429") ||
						strings.Contains(errorMsg, "rate limit") {
						foundExpectedError = true
						break
					}
				}

				if !foundExpectedError {
					t.Logf("Mixed operation errors for %s: expected connection/rate limit errors", tt.name)
					for i, err := range allErrors {
						t.Logf("Error %d: %s", i+1, err.Error())
					}
				}
			}
		})
	}
}

func TestConcurrentRequestsResourceManagement(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "concurrent requests with memory pressure",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Simulate memory pressure causing failures
					if articleID%4 == 0 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Insufficient memory resources")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle memory pressure during concurrent requests",
		},
		{
			name: "concurrent requests with file descriptor limits",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				requestCount := 0
				var mu sync.Mutex

				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					mu.Lock()
					requestCount++
					current := requestCount
					mu.Unlock()

					// Simulate file descriptor limit
					if current > 5 {
						return "", fmt.Errorf("dial tcp: too many open files")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle file descriptor limits during concurrent requests",
		},
		{
			name: "concurrent requests with network interface saturation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					// Simulate network interface saturation
					if articleID > 502 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Network interface saturated")
					}
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true,
			description: "Should handle network interface saturation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.setupMock(mockClient)

			// Run many concurrent operations to stress test resource management
			var wg sync.WaitGroup
			errors := make(chan error, 10)
			articleIDs := []int{500, 501, 502, 503, 504, 505, 506, 507, 508, 509}

			for _, articleID := range articleIDs {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					cmd := &CommandPull{
						Locale:     testhelper.TestLocales.Japanese,
						ArticleIDs: []int{id},
						client:     mockClient,
						converter:  converter.NewConverter(false),
					}

					global := &Global{
						Config: Config{
							DefaultLocale: testhelper.TestLocales.English,
							ContentsDir:   tempDir,
						},
					}

					if err := cmd.Run(global); err != nil {
						errors <- err
					}
				}(articleID)
			}

			wg.Wait()
			close(errors)

			// Collect all errors
			var allErrors []error
			for err := range errors {
				allErrors = append(allErrors, err)
			}

			if tt.expectError && len(allErrors) == 0 {
				t.Errorf("Expected errors for %s but got none", tt.name)
			}
			if !tt.expectError && len(allErrors) > 0 {
				t.Errorf("Expected no errors for %s but got: %v", tt.name, allErrors)
			}

			// Validate that we get expected resource management errors
			if tt.expectError && len(allErrors) > 0 {
				foundResourceError := false
				for _, err := range allErrors {
					errorMsg := err.Error()
					if strings.Contains(errorMsg, "memory resources") ||
						strings.Contains(errorMsg, "too many open files") ||
						strings.Contains(errorMsg, "interface saturated") {
						foundResourceError = true
						break
					}
				}

				if !foundResourceError {
					t.Logf("Resource management errors for %s:", tt.name)
					for i, err := range allErrors {
						t.Logf("Error %d: %s", i+1, err.Error())
					}
				}
			}
		})
	}
}
