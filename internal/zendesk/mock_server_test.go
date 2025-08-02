package zendesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAdvancedMockServer_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *MockServerConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &MockServerConfig{
				BaseLatency:   50 * time.Millisecond,
				ErrorRate:     0.1,
				RateLimit:     500,
				EnableLogging: true,
				StrictMode:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := NewAdvancedMockServer(tt.config)
			defer server.Close()

			// Verify server is running
			if server.URL() == "" {
				t.Error("Server URL should not be empty")
			}

			// Verify basic HTTP connectivity
			resp, err := http.Get(server.URL() + "/api/v2/help_center/en_us/articles/456.json")
			if err != nil {
				t.Fatalf("Failed to connect to mock server: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestAdvancedMockServer_ArticleOperations(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	baseURL := server.URL()

	t.Run("ShowArticle", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]*Article
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		article := result["article"]
		if article == nil {
			t.Error("Article should not be nil")
		}

		if article.ID != 456 {
			t.Errorf("Expected article ID 456, got %d", article.ID)
		}

		if article.Title != "Sample Article" {
			t.Errorf("Expected title 'Sample Article', got '%s'", article.Title)
		}
	})

	t.Run("ShowArticle_NotFound", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/999.json")
		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("CreateArticle", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v2/help_center/en_us/sections/123/articles.json", "application/json", strings.NewReader(`{"article":{"title":"New Article"}}`))
		if err != nil {
			t.Fatalf("Failed to create article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]*Article
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		article := result["article"]
		if article == nil {
			t.Error("Created article should not be nil")
		}

		if article.ID <= 0 {
			t.Errorf("Created article should have positive ID, got %d", article.ID)
		}

		if article.SectionID != 123 {
			t.Errorf("Expected section ID 123, got %d", article.SectionID)
		}
	})

	t.Run("CreateArticle_InvalidSection", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v2/help_center/en_us/sections/999/articles.json", "application/json", strings.NewReader(`{"article":{"title":"New Article"}}`))
		if err != nil {
			t.Fatalf("Failed to create article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid section, got %d", resp.StatusCode)
		}
	})
}

func TestAdvancedMockServer_TranslationOperations(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	baseURL := server.URL()

	t.Run("ShowTranslation", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v2/help_center/articles/456/translations/ja")
		if err != nil {
			t.Fatalf("Failed to get translation: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]*Translation
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		translation := result["translation"]
		if translation == nil {
			t.Error("Translation should not be nil")
		}

		if translation.SourceID != 456 {
			t.Errorf("Expected source ID 456, got %d", translation.SourceID)
		}

		if translation.Locale != "ja" {
			t.Errorf("Expected locale 'ja', got '%s'", translation.Locale)
		}
	})

	t.Run("ShowTranslation_NotFound", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v2/help_center/articles/999/translations/ja")
		if err != nil {
			t.Fatalf("Failed to get translation: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("CreateTranslation", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v2/help_center/articles/456/translations.json", "application/json", strings.NewReader(`{"translation":{"locale":"fr","title":"Article Français"}}`))
		if err != nil {
			t.Fatalf("Failed to create translation: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]*Translation
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		translation := result["translation"]
		if translation == nil {
			t.Error("Created translation should not be nil")
		}

		if translation.SourceID != 456 {
			t.Errorf("Expected source ID 456, got %d", translation.SourceID)
		}
	})

	t.Run("CreateTranslation_InvalidArticle", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v2/help_center/articles/999/translations.json", "application/json", strings.NewReader(`{"translation":{"locale":"fr","title":"Test"}}`))
		if err != nil {
			t.Fatalf("Failed to create translation: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid article, got %d", resp.StatusCode)
		}
	})
}

func TestAdvancedMockServer_ScenarioManagement(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	baseURL := server.URL()

	t.Run("NormalScenario", func(t *testing.T) {
		// Should start with Normal scenario
		if scenario := server.scenarios.GetScenario(); scenario != "Normal" {
			t.Errorf("Expected default scenario 'Normal', got '%s'", scenario)
		}

		// Normal requests should succeed
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 in Normal scenario, got %d", resp.StatusCode)
		}
	})

	t.Run("AuthFailureScenario", func(t *testing.T) {
		// Switch to AuthFailure scenario
		if err := server.SetScenario("AuthFailure"); err != nil {
			t.Fatalf("Failed to set AuthFailure scenario: %v", err)
		}

		// All requests should fail with 401
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 in AuthFailure scenario, got %d", resp.StatusCode)
		}

		// Switch back to Normal
		if err := server.SetScenario("Normal"); err != nil {
			t.Fatalf("Failed to switch back to Normal scenario: %v", err)
		}
	})

	t.Run("HighLatencyScenario", func(t *testing.T) {
		// Switch to HighLatency scenario
		if err := server.SetScenario("HighLatency"); err != nil {
			t.Fatalf("Failed to set HighLatency scenario: %v", err)
		}

		// Measure response time
		start := time.Now()
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get article: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Should take at least the minimum latency
		if duration < 200*time.Millisecond {
			t.Errorf("Expected latency >= 200ms, got %v", duration)
		}
	})

	t.Run("InvalidScenario", func(t *testing.T) {
		// Try to set invalid scenario
		if err := server.SetScenario("NonExistentScenario"); err == nil {
			t.Error("Expected error when setting invalid scenario")
		}
	})
}

func TestAdvancedMockServer_RequestLogging(t *testing.T) {
	t.Parallel()

	config := &MockServerConfig{
		EnableLogging: true,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	// Clear any existing logs
	server.ClearRequestLog()

	// Make some requests
	http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
	http.Post(baseURL+"/api/v2/help_center/en_us/sections/123/articles.json", "application/json", strings.NewReader(`{"article":{"title":"Test"}}`))

	// Check logs
	logs := server.GetRequestLog()
	if len(logs) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(logs))
	}

	// Verify log structure
	if len(logs) > 0 {
		log := logs[0]
		if log.Method == "" {
			t.Error("Log should contain method")
		}
		if log.Path == "" {
			t.Error("Log should contain path")
		}
		if log.Timestamp.IsZero() {
			t.Error("Log should contain timestamp")
		}
	}
}

func TestAdvancedMockServer_DataStoreOperations(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	// Test data store statistics
	stats := server.dataStore.GetStats()
	if stats["articles"] != 1 { // Should have default article
		t.Errorf("Expected 1 article, got %d", stats["articles"])
	}

	if stats["translations"] != 1 { // Should have default translation
		t.Errorf("Expected 1 translation, got %d", stats["translations"])
	}

	if stats["sections"] < 3 { // Should have default sections
		t.Errorf("Expected at least 3 sections, got %d", stats["sections"])
	}

	// Test relationship validation
	issues := server.dataStore.ValidateRelationships()
	if len(issues) > 0 {
		t.Errorf("Data integrity issues found: %v", issues)
	}

	// Test backup and restore
	backup := server.dataStore.Backup()
	if backup == nil {
		t.Error("Backup should not be nil")
	}

	if backup.Timestamp.IsZero() {
		t.Error("Backup should have timestamp")
	}

	// Modify data and restore
	server.dataStore.Reset()
	server.dataStore.Restore(backup)

	// Verify restoration
	newStats := server.dataStore.GetStats()
	if newStats["articles"] != stats["articles"] {
		t.Errorf("Article count mismatch after restore: got %d, want %d", newStats["articles"], stats["articles"])
	}
}

func TestAdvancedMockServer_ErrorHandling(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	baseURL := server.URL()

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Invalid URL format",
			method:         "GET",
			url:            "/api/v2/help_center/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid article ID",
			method:         "GET",
			url:            "/api/v2/help_center/en_us/articles/invalid.json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid section ID",
			method:         "POST",
			url:            "/api/v2/help_center/en_us/sections/invalid/articles.json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			switch tt.method {
			case "GET":
				resp, err = http.Get(baseURL + tt.url)
			case "POST":
				resp, err = http.Post(baseURL+tt.url, "application/json", strings.NewReader(`{}`))
			default:
				t.Fatalf("Unsupported method: %s", tt.method)
			}

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestAdvancedMockServer_StatefulBehavior(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	baseURL := server.URL()

	// Create a new article
	createResp, err := http.Post(baseURL+"/api/v2/help_center/en_us/sections/123/articles.json", "application/json", strings.NewReader(`{"article":{"title":"Stateful Test"}}`))
	if err != nil {
		t.Fatalf("Failed to create article: %v", err)
	}
	defer createResp.Body.Close()

	var createResult map[string]*Article
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	articleID := createResult["article"].ID

	// Retrieve the created article
	getResp, err := http.Get(fmt.Sprintf("%s/api/v2/help_center/en_us/articles/%d.json", baseURL, articleID))
	if err != nil {
		t.Fatalf("Failed to get created article: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 when retrieving created article, got %d", getResp.StatusCode)
	}

	var getResult map[string]*Article
	if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
		t.Fatalf("Failed to decode get response: %v", err)
	}

	retrievedArticle := getResult["article"]
	if retrievedArticle.ID != articleID {
		t.Errorf("Retrieved article ID mismatch: got %d, want %d", retrievedArticle.ID, articleID)
	}

	if !strings.Contains(retrievedArticle.Title, fmt.Sprintf("Article %d", articleID)) {
		t.Errorf("Retrieved article title unexpected: got '%s'", retrievedArticle.Title)
	}
}

func TestAdvancedMockServer_ErrorSimulation(t *testing.T) {
	t.Parallel()

	config := &MockServerConfig{
		EnableErrorSim:  true,
		ErrorScenarios:  []string{"AuthenticationFailures", "RateLimiting"},
		EnableLogging:   true,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	t.Run("AuthenticationFailures", func(t *testing.T) {
		// Clear error tracking
		server.ResetErrorTracking()

		// Make requests without proper authentication
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", baseURL+"/api/v2/help_center/en_us/articles/456.json", nil)
			// Don't set Authorization header to trigger auth failure
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()

			// Should sometimes return 401 due to error simulation
			if resp.StatusCode == http.StatusUnauthorized {
				break // Found at least one auth failure
			}
		}

		// Check error distributions
		distributions := server.GetErrorDistributions()
		if len(distributions) == 0 {
			t.Error("Expected error tracking data")
		}
	})

	t.Run("RateLimiting", func(t *testing.T) {
		server.ResetErrorTracking()

		// Make POST requests that should trigger rate limiting
		for i := 0; i < 5; i++ {
			resp, err := http.Post(baseURL+"/api/v2/help_center/en_us/sections/123/articles.json", "application/json", strings.NewReader(`{"article":{"title":"Test"}}`))
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				// Verify rate limit headers
				if resp.Header.Get("Retry-After") == "" {
					t.Error("Expected Retry-After header for rate limit response")
				}
				break
			}
		}
	})

	t.Run("CustomErrorScenario", func(t *testing.T) {
		// Add a custom error scenario
		customScenario := &ErrorScenario{
			Name:        "TestScenario",
			Probability: 1.0,
			Errors: []ErrorDefinition{
				{
					StatusCode:  http.StatusTeapot,
					ErrorType:   "CustomError",
					Description: "This is a test error",
					Condition: func(r *http.Request) bool {
						return strings.Contains(r.URL.Path, "test")
					},
				},
			},
		}

		server.AddCustomErrorScenario("TestScenario", customScenario)

		// Enable the custom scenario
		server.EnableErrorSimulation([]string{"TestScenario"})

		// Make a request that should trigger the custom error
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/test.json")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("Expected custom error status %d, got %d", http.StatusTeapot, resp.StatusCode)
		}
	})
}

func TestAdvancedMockServer_ErrorSimulationDisabled(t *testing.T) {
	t.Parallel()

	config := &MockServerConfig{
		EnableErrorSim: false, // Disabled
		EnableLogging:  false,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	// Make multiple requests - should all succeed when error simulation is disabled
	for i := 0; i < 10; i++ {
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 with error simulation disabled, got %d", resp.StatusCode)
		}
	}

	// Error distributions should be empty
	distributions := server.GetErrorDistributions()
	if len(distributions) > 0 {
		t.Error("Expected no error tracking when error simulation is disabled")
	}
}

func TestAdvancedMockServer_CompositeErrorScenario(t *testing.T) {
	t.Parallel()

	server := NewAdvancedMockServer(nil)
	defer server.Close()

	// Create a composite scenario combining multiple error types
	server.CreateCompositeErrorScenario("CompositeTest", []string{"AuthenticationFailures", "ValidationErrors"}, 0.5)

	// Enable the composite scenario
	server.EnableErrorSimulation([]string{"CompositeTest"})

	baseURL := server.URL()

	// Make requests that could trigger either auth or validation errors
	errorTypes := make(map[int]bool)
	
	for i := 0; i < 20; i++ {
		resp, err := http.Post(baseURL+"/api/v2/help_center/en_us/sections/123/articles.json", "application/json", strings.NewReader(`{"article":{"title":"Test"}}`))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			errorTypes[resp.StatusCode] = true
		}
	}

	// Should have encountered different types of errors from the composite scenario
	if len(errorTypes) == 0 {
		t.Log("No errors encountered in composite scenario test (this is possible but unlikely)")
	}
}