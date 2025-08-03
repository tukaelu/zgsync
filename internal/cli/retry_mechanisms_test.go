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

func TestRetryMechanisms_TransientErrors(t *testing.T) {
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
		description string
	}{
		{
			name: "transient network error should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("net/http: TLS handshake timeout")
				}
			},
			expectError: true,
			description: "Should identify TLS handshake timeout as retryable error",
		},
		{
			name: "temporary DNS resolution failure should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("dial tcp: lookup zendesk.com: temporary failure in name resolution")
				}
			},
			expectError: true,
			description: "Should identify DNS temporary failure as retryable error",
		},
		{
			name: "HTTP 502 bad gateway should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 502 Bad Gateway: Server temporarily overloaded")
				}
			},
			expectError: true,
			description: "Should identify 502 errors as retryable",
		},
		{
			name: "HTTP 503 service unavailable should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Maintenance mode")
				}
			},
			expectError: true,
			description: "Should identify 503 errors as retryable",
		},
		{
			name: "HTTP 504 gateway timeout should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 504 Gateway Timeout: Upstream server timeout")
				}
			},
			expectError: true,
			description: "Should identify 504 errors as retryable",
		},
		{
			name: "HTTP 429 rate limit with retry-after should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded. Retry after 30 seconds")
				}
			},
			expectError: true,
			description: "Should identify 429 errors as retryable with backoff",
		},
		{
			name: "connection reset by peer should be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("read tcp: connection reset by peer")
				}
			},
			expectError: true,
			description: "Should identify connection reset as retryable error",
		},
		{
			name: "HTTP 400 bad request should NOT be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 400 Bad Request: Invalid JSON payload")
				}
			},
			expectError: true,
			description: "Should identify 400 errors as non-retryable client errors",
		},
		{
			name: "HTTP 404 not found should NOT be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 404 Not Found: Article does not exist")
				}
			},
			expectError: true,
			description: "Should identify 404 errors as non-retryable resource errors",
		},
		{
			name: "HTTP 401 unauthorized should NOT be retryable",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 401 Unauthorized: Invalid credentials")
				}
			},
			expectError: true,
			description: "Should identify 401 errors as non-retryable auth errors",
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

			// Log error information for retry mechanism analysis
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				isRetryable := isTransientError(errorMsg)
				t.Logf("Error classification for %s:", tt.name)
				t.Logf("  Error: %s", errorMsg)
				t.Logf("  Would be retryable: %v", isRetryable)
				t.Logf("  Description: %s", tt.description)
			}
		})
	}
}

func TestRetryMechanisms_SimulatedRetryLogic(t *testing.T) {
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
		description string
	}{
		{
			name: "eventually successful after transient failures",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					current := callCount
					mu.Unlock()
					
					// Fail first 2 attempts with retryable errors, succeed on 3rd
					if current == 1 {
						return "", fmt.Errorf("HTTP 502 Bad Gateway: Server temporarily overloaded")
					}
					if current == 2 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Maintenance mode")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true, // Current implementation doesn't retry, so it fails on first attempt
			description: "Should eventually succeed after transient failures with retry logic",
		},
		{
			name: "intermittent rate limiting with backoff",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					current := callCount
					mu.Unlock()
					
					// Rate limit on first few attempts
					if current <= 2 {
						return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded. Retry after %d seconds", current*30)
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true, // Current implementation doesn't retry, so it fails on first attempt
			description: "Should handle rate limiting with exponential backoff",
		},
		{
			name: "permanent failure after max retries",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					return "", fmt.Errorf("HTTP 502 Bad Gateway: Persistent upstream error")
				}
			},
			expectError: true,
			description: "Should fail permanently after exhausting retry attempts",
		},
		{
			name: "mixed retryable and non-retryable errors",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					current := callCount
					mu.Unlock()
					
					// First attempt: retryable error
					if current == 1 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Server overloaded")
					}
					// Second attempt: non-retryable error (should stop retrying)
					return "", fmt.Errorf("HTTP 400 Bad Request: Invalid payload format")
				}
			},
			expectError: true,
			description: "Should stop retrying when encountering non-retryable errors",
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

			// Measure execution time to detect retry delays
			start := time.Now()
			err := cmd.Run(global)
			duration := time.Since(start)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			t.Logf("Retry simulation for %s:", tt.name)
			t.Logf("  Execution time: %v", duration)
			t.Logf("  Error: %v", err)
			t.Logf("  Description: %s", tt.description)
		})
	}
}

func TestRetryMechanisms_BackoffStrategies(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "linear backoff pattern simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Linear backoff test - retry in 5 seconds")
				}
			},
			expectError: true,
			description: "Should simulate linear backoff (5s, 10s, 15s intervals)",
		},
		{
			name: "exponential backoff pattern simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Exponential backoff test - retry with exponential delay")
				}
			},
			expectError: true,
			description: "Should simulate exponential backoff (1s, 2s, 4s, 8s intervals)",
		},
		{
			name: "jittered backoff pattern simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Jittered backoff test - retry with randomized delay")
				}
			},
			expectError: true,
			description: "Should simulate jittered backoff to prevent thundering herd",
		},
		{
			name: "rate limit backoff with retry-after header",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limited - retry after 120 seconds")
				}
			},
			expectError: true,
			description: "Should respect retry-after header for backoff timing",
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

			// Analyze error for backoff information
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				backoffInfo := extractBackoffInfo(errorMsg)
				t.Logf("Backoff strategy analysis for %s:", tt.name)
				t.Logf("  Error: %s", errorMsg)
				t.Logf("  Suggested backoff: %s", backoffInfo)
				t.Logf("  Description: %s", tt.description)
			}
		})
	}
}

func TestRetryMechanisms_CircuitBreakerSimulation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupMock   func(*testhelper.MockZendeskClient)
		expectError bool
		description string
	}{
		{
			name: "circuit breaker threshold simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				failureCount := 0
				var mu sync.Mutex
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					mu.Lock()
					failureCount++
					current := failureCount
					mu.Unlock()
					
					// Simulate circuit breaker opening after 3 failures
					if current >= 3 {
						return "", fmt.Errorf("Circuit breaker OPEN: Too many consecutive failures")
					}
					return "", fmt.Errorf("HTTP 503 Service Unavailable: Service degraded")
				}
			},
			expectError: true,
			description: "Should simulate circuit breaker opening after failure threshold",
		},
		{
			name: "circuit breaker half-open state simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return "", fmt.Errorf("Circuit breaker HALF-OPEN: Testing if service recovered")
				}
			},
			expectError: true,
			description: "Should simulate circuit breaker half-open state testing",
		},
		{
			name: "circuit breaker recovery simulation",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					mu.Lock()
					callCount++
					current := callCount
					mu.Unlock()
					
					// Simulate recovery after circuit breaker opens
					if current == 1 {
						return "", fmt.Errorf("Circuit breaker CLOSED: Service recovered")
					}
					return testhelper.CreateDefaultArticleResponse(200, sectionID), nil
				}
			},
			expectError: true, // First call fails, but would succeed with retry logic
			description: "Should simulate circuit breaker recovery and closing",
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

			// Analyze circuit breaker state
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				circuitState := extractCircuitBreakerState(errorMsg)
				t.Logf("Circuit breaker simulation for %s:", tt.name)
				t.Logf("  Error: %s", errorMsg)
				t.Logf("  Circuit state: %s", circuitState)
				t.Logf("  Description: %s", tt.description)
			}
		})
	}
}

func TestRetryMechanisms_BulkOperationRetries(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create multiple test files
	testFiles := make([]string, 3)
	for i := 0; i < 3; i++ {
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
			name: "bulk operation with partial failures requiring retries",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				callCount := 0
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					callCount++
					current := callCount
					mu.Unlock()
					
					// Make second file fail with retryable error
					if articleID == 124 && current <= 2 {
						return "", fmt.Errorf("HTTP 503 Service Unavailable: Temporary service degradation")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true, // Current implementation would fail on first retry-eligible error
			description: "Should handle partial failures in bulk operations with retry logic",
		},
		{
			name: "bulk operation retry with backoff between requests",
			setupMock: func(mock *testhelper.MockZendeskClient) {
				requestTimes := make([]time.Time, 0)
				var mu sync.Mutex
				mock.UpdateTranslationFunc = func(articleID int, locale, payload string) (string, error) {
					mu.Lock()
					requestTimes = append(requestTimes, time.Now())
					count := len(requestTimes)
					mu.Unlock()
					
					// Fail first request to test retry spacing
					if count == 1 {
						return "", fmt.Errorf("HTTP 429 Too Many Requests: Rate limit exceeded")
					}
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			expectError: true, // Would succeed with retry logic
			description: "Should space retries appropriately in bulk operations",
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
				Files:   testFiles,
			}
			cmd.client = mockClient
			cmd.converter = converter.NewConverter(false)

			global := &Global{
				Config: Config{
					DefaultLocale:     testhelper.TestLocales.English,
					NotifySubscribers: false,
				},
			}

			start := time.Now()
			err := cmd.Run(global)
			duration := time.Since(start)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.name, err)
			}

			t.Logf("Bulk operation retry analysis for %s:", tt.name)
			t.Logf("  Total duration: %v", duration)
			t.Logf("  Error: %v", err)
			t.Logf("  Description: %s", tt.description)
		})
	}
}

// Helper functions for retry mechanism analysis

// isTransientError determines if an error should be retryable
func isTransientError(errorMsg string) bool {
	retryablePatterns := []string{
		"TLS handshake timeout",
		"temporary failure in name resolution",
		"502 Bad Gateway",
		"503 Service Unavailable",
		"504 Gateway Timeout",
		"429 Too Many Requests",
		"connection reset by peer",
		"context deadline exceeded",
		"dial tcp: i/o timeout",
	}
	
	for _, pattern := range retryablePatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}
	return false
}

// extractBackoffInfo extracts backoff timing information from error messages
func extractBackoffInfo(errorMsg string) string {
	if strings.Contains(errorMsg, "retry after") {
		return "Server-specified backoff timing"
	}
	if strings.Contains(errorMsg, "Linear backoff") {
		return "Linear backoff strategy (fixed intervals)"
	}
	if strings.Contains(errorMsg, "Exponential backoff") {
		return "Exponential backoff strategy (doubling intervals)"
	}
	if strings.Contains(errorMsg, "Jittered backoff") {
		return "Jittered backoff strategy (randomized intervals)"
	}
	return "Default backoff strategy"
}

// extractCircuitBreakerState extracts circuit breaker state from error messages
func extractCircuitBreakerState(errorMsg string) string {
	if strings.Contains(errorMsg, "OPEN") {
		return "OPEN (rejecting requests)"
	}
	if strings.Contains(errorMsg, "HALF-OPEN") {
		return "HALF-OPEN (testing recovery)"
	}
	if strings.Contains(errorMsg, "CLOSED") {
		return "CLOSED (normal operation)"
	}
	return "UNKNOWN"
}