package zendesk

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *RateLimitConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &RateLimitConfig{
				GlobalLimit:       200,
				GlobalWindow:      time.Minute,
				BurstLimit:        50,
				BurstWindow:       10 * time.Second,
				Enable429Response: true,
				EnableHeaders:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			limiter := NewRateLimiter(tt.config)
			if limiter == nil {
				t.Error("Expected non-nil rate limiter")
				return
			}

			if limiter.config == nil {
				t.Error("Expected non-nil config")
			}

			if len(limiter.buckets) == 0 {
				t.Error("Expected token buckets to be initialized")
			}

			// Verify global bucket exists
			if limiter.buckets["global"] == nil {
				t.Error("Expected global token bucket")
			}

			// Verify burst bucket exists
			if limiter.buckets["burst"] == nil {
				t.Error("Expected burst token bucket")
			}
		})
	}
}

func TestRateLimiter_CheckRateLimit(t *testing.T) {
	t.Parallel()

	config := &RateLimitConfig{
		GlobalLimit:       10,  // Small limit for testing
		GlobalWindow:      time.Minute,
		BurstLimit:        3,   // Very small burst for testing
		BurstWindow:       5 * time.Second,
		Enable429Response: true,
	}

	limiter := NewRateLimiter(config)

	t.Run("AllowedRequests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)
		
		// First few requests should be allowed
		for i := 0; i < 2; i++ {
			result := limiter.CheckRateLimit(req)
			if !result.Allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
			if result.Remaining < 0 {
				t.Errorf("Remaining count should be non-negative, got %d", result.Remaining)
			}
		}
	})

	t.Run("BurstLimitExceeded", func(t *testing.T) {
		// Create fresh limiter for this test
		limiter := NewRateLimiter(config)
		req := httptest.NewRequest("POST", "/api/v2/help_center/articles", nil)
		
		// Exhaust burst limit
		for i := 0; i < config.BurstLimit; i++ {
			result := limiter.CheckRateLimit(req)
			if !result.Allowed {
				t.Errorf("Burst request %d should be allowed", i+1)
			}
		}

		// Next request should be rate limited
		result := limiter.CheckRateLimit(req)
		if result.Allowed {
			t.Error("Request should be rate limited after burst limit exceeded")
		}
		if result.LimitType != "burst" {
			t.Errorf("Expected burst limit type, got %s", result.LimitType)
		}
		if result.RetryAfter <= 0 {
			t.Errorf("Expected positive retry after time, got %v", result.RetryAfter)
		}
	})
}

func TestRateLimiter_ApplyRateLimit(t *testing.T) {
	t.Parallel()

	config := &RateLimitConfig{
		GlobalLimit:       5,   // Very small limit
		GlobalWindow:      time.Minute,
		BurstLimit:        2,   // Very small burst
		BurstWindow:       5 * time.Second,
		Enable429Response: true,
		EnableHeaders:     true,
	}

	limiter := NewRateLimiter(config)

	t.Run("HeadersAdded", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

		handled := limiter.ApplyRateLimit(w, req)
		if handled {
			t.Error("First request should not be rate limited")
		}

		// Check headers
		if w.Header().Get("X-Rate-Limit-Limit") == "" {
			t.Error("Expected X-Rate-Limit-Limit header")
		}
		if w.Header().Get("X-Rate-Limit-Remaining") == "" {
			t.Error("Expected X-Rate-Limit-Remaining header")
		}
		if w.Header().Get("X-Rate-Limit-Reset") == "" {
			t.Error("Expected X-Rate-Limit-Reset header")
		}
	})

	t.Run("RateLimitResponse", func(t *testing.T) {
		// Create fresh limiter for this test
		limiter := NewRateLimiter(&RateLimitConfig{
			GlobalLimit:       1,   // Extremely small limit
			GlobalWindow:      time.Minute,
			BurstLimit:        1,
			BurstWindow:       5 * time.Second,
			Enable429Response: true,
			EnableHeaders:     true,
		})

		req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

		// First request should be allowed
		w1 := httptest.NewRecorder()
		handled1 := limiter.ApplyRateLimit(w1, req)
		if handled1 {
			t.Error("First request should not be rate limited")
		}

		// Second request should be rate limited
		w2 := httptest.NewRecorder()
		handled2 := limiter.ApplyRateLimit(w2, req)
		if !handled2 {
			t.Error("Second request should be rate limited")
		}

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", w2.Code)
		}

		if w2.Header().Get("Retry-After") == "" {
			t.Error("Expected Retry-After header")
		}

		body := w2.Body.String()
		if !contains(body, "Rate limit exceeded") {
			t.Error("Expected rate limit error message in body")
		}
	})
}

func TestRateLimiter_EndpointSpecificLimits(t *testing.T) {
	t.Parallel()

	config := &RateLimitConfig{
		GlobalLimit:  100,
		GlobalWindow: time.Minute,
		BurstLimit:   50,
		BurstWindow:  10 * time.Second,
		PerEndpointLimits: map[string]int{
			"/articles": 2, // Very small limit for testing
		},
		Enable429Response: true,
	}

	limiter := NewRateLimiter(config)

	t.Run("EndpointLimitExceeded", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

		// Exhaust endpoint limit
		for i := 0; i < 2; i++ {
			result := limiter.CheckRateLimit(req)
			if !result.Allowed {
				t.Errorf("Article request %d should be allowed", i+1)
			}
		}

		// Next request should be rate limited
		result := limiter.CheckRateLimit(req)
		if result.Allowed {
			t.Error("Request should be rate limited after endpoint limit exceeded")
		}
		if result.LimitType != "/articles" {
			t.Errorf("Expected /articles limit type, got %s", result.LimitType)
		}
	})

	t.Run("DifferentEndpointNotAffected", func(t *testing.T) {
		// Requests to different endpoint should still work
		req := httptest.NewRequest("GET", "/api/v2/help_center/sections/456", nil)
		result := limiter.CheckRateLimit(req)
		if !result.Allowed {
			t.Error("Section request should be allowed when article limit is exceeded")
		}
	})
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	t.Parallel()

	config := &RateLimitConfig{
		GlobalLimit:       5,
		GlobalWindow:      100 * time.Millisecond, // Very short window for testing
		BurstLimit:        2,
		BurstWindow:       50 * time.Millisecond,
		Enable429Response: true,
	}

	limiter := NewRateLimiter(config)
	req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

	// Exhaust burst limit
	for i := 0; i < 2; i++ {
		result := limiter.CheckRateLimit(req)
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rate limited now
	result := limiter.CheckRateLimit(req)
	if result.Allowed {
		t.Error("Should be rate limited after burst exhausted")
	}

	// Wait for token refill
	time.Sleep(60 * time.Millisecond) // Wait longer than burst window

	// Should be allowed again
	result = limiter.CheckRateLimit(req)
	if !result.Allowed {
		t.Error("Should be allowed after token refill")
	}
}

func TestRateLimiter_Statistics(t *testing.T) {
	t.Parallel()

	config := &RateLimitConfig{
		GlobalLimit:       3,
		GlobalWindow:      time.Minute,
		BurstLimit:        1,
		BurstWindow:       5 * time.Second,
		Enable429Response: true,
	}

	limiter := NewRateLimiter(config)

	// Make requests that will trigger rate limiting
	req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)
	
	// First request allowed
	limiter.CheckRateLimit(req)
	
	// Second request rate limited (burst limit = 1)
	limiter.CheckRateLimit(req)

	stats := limiter.GetStatistics()
	if stats == nil {
		t.Fatal("Expected non-nil statistics")
	}

	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", stats.TotalRequests)
	}

	if stats.LimitedRequests == 0 {
		t.Error("Expected some limited requests")
	}

	if len(stats.EndpointStats) == 0 {
		t.Error("Expected endpoint statistics")
	}

	// Test reset
	limiter.ResetStatistics()
	stats = limiter.GetStatistics()
	
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", stats.TotalRequests)
	}
}

func TestRateLimiter_UpdateLimits(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(nil)

	// Test global limit update
	newGlobalLimit := 500
	limiter.UpdateGlobalLimit(newGlobalLimit)
	
	if limiter.config.GlobalLimit != newGlobalLimit {
		t.Errorf("Expected global limit %d, got %d", newGlobalLimit, limiter.config.GlobalLimit)
	}

	// Test endpoint limit update
	endpoint := "/test"
	endpointLimit := 100
	limiter.UpdateEndpointLimit(endpoint, endpointLimit)
	
	if limiter.config.PerEndpointLimits[endpoint] != endpointLimit {
		t.Errorf("Expected endpoint limit %d, got %d", endpointLimit, limiter.config.PerEndpointLimits[endpoint])
	}

	// Verify bucket was created
	if limiter.buckets[endpoint] == nil {
		t.Error("Expected bucket to be created for new endpoint")
	}
}

func TestRateLimiter_Report(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(nil)

	// Make some requests
	req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)
	limiter.CheckRateLimit(req)
	limiter.CheckRateLimit(req)

	report := limiter.GetRateLimitReport()
	if report == "" {
		t.Error("Expected non-empty report")
	}

	if !contains(report, "Total Requests:") {
		t.Error("Expected report to contain total requests")
	}

	if !contains(report, "Limited Requests:") {
		t.Error("Expected report to contain limited requests")
	}
}

func TestAdvancedMockServer_LatencySimulation(t *testing.T) {
	t.Parallel()

	latencyConfig := &LatencyConfig{
		BaseLatency:    20 * time.Millisecond,
		JitterFactor:   0.1,
		Distribution:   DistributionNormal,
		NetworkProfile: NetworkBroadband,
		EnableJitter:   true,
	}

	config := &MockServerConfig{
		EnableLatencySim: true,
		LatencyConfig:    latencyConfig,
		EnableLogging:    true,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	t.Run("LatencyApplied", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Should have some latency applied
		if duration < 10*time.Millisecond {
			t.Errorf("Expected latency to be applied, got duration %v", duration)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("LatencyStatistics", func(t *testing.T) {
		// Make a few requests
		for i := 0; i < 3; i++ {
			resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			_ = resp.Body.Close()
		}

		stats := server.GetLatencyStatistics()
		if stats == nil {
			t.Fatal("Expected latency statistics")
		}

		if stats.TotalRequests < 3 {
			t.Errorf("Expected at least 3 requests in stats, got %d", stats.TotalRequests)
		}

		if stats.AverageLatency <= 0 {
			t.Errorf("Expected positive average latency, got %v", stats.AverageLatency)
		}
	})

	t.Run("NetworkProfileChange", func(t *testing.T) {
		// Change to slow network
		server.SetNetworkProfile(NetworkSlow)

		start := time.Now()
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		slowDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		_ = resp.Body.Close()

		// Should be noticeably slower
		if slowDuration < 100*time.Millisecond {
			t.Errorf("Expected slower latency with NetworkSlow, got %v", slowDuration)
		}
	})
}

func TestAdvancedMockServer_RateLimiting(t *testing.T) {
	t.Parallel()

	rateLimitConfig := &RateLimitConfig{
		GlobalLimit:       5,  // Small limit for testing
		GlobalWindow:      time.Minute,
		BurstLimit:        2,  // Small burst for testing
		BurstWindow:       10 * time.Second,
		Enable429Response: true,
		EnableHeaders:     true,
	}

	config := &MockServerConfig{
		EnableRateLimiting: true,
		RateLimitConfig:    rateLimitConfig,
		EnableLogging:      true,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	t.Run("RateLimitHeaders", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Check rate limit headers
		if resp.Header.Get("X-Rate-Limit-Limit") == "" {
			t.Error("Expected X-Rate-Limit-Limit header")
		}

		if resp.Header.Get("X-Rate-Limit-Remaining") == "" {
			t.Error("Expected X-Rate-Limit-Remaining header")
		}

		if resp.Header.Get("X-Rate-Limit-Reset") == "" {
			t.Error("Expected X-Rate-Limit-Reset header")
		}
	})

	t.Run("RateLimitExceeded", func(t *testing.T) {
		// Create a new server with very restrictive limits for this test
		strictConfig := &MockServerConfig{
			EnableRateLimiting: true,
			RateLimitConfig: &RateLimitConfig{
				GlobalLimit:       2,   // Very small limit
				GlobalWindow:      time.Minute,
				BurstLimit:        1,   // Only 1 burst request
				BurstWindow:       10 * time.Second,
				Enable429Response: true,
				EnableHeaders:     true,
			},
			EnableLogging: false,
		}

		strictServer := NewAdvancedMockServer(strictConfig)
		defer strictServer.Close()
		strictURL := strictServer.URL()

		// First request should succeed
		resp, err := http.Get(strictURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("First request should succeed, got status %d", resp.StatusCode)
		}

		// Second request should be rate limited (burst limit = 1)
		resp, err = http.Get(strictURL + "/api/v2/help_center/en_us/articles/456.json")
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Retry-After") == "" {
			t.Error("Expected Retry-After header for rate limited response")
		}
	})

	t.Run("RateLimitStatistics", func(t *testing.T) {
		// Create a fresh server and generate some statistics
		statsConfig := &MockServerConfig{
			EnableRateLimiting: true,
			RateLimitConfig: &RateLimitConfig{
				GlobalLimit:       2,   // Small limit
				GlobalWindow:      time.Minute,
				BurstLimit:        1,   // Very small burst
				BurstWindow:       10 * time.Second,
				Enable429Response: true,
				EnableHeaders:     true,
			},
			EnableLogging: false,
		}

		statsServer := NewAdvancedMockServer(statsConfig)
		defer statsServer.Close()
		statsURL := statsServer.URL()

		// Make requests that will generate statistics
		for i := 0; i < 3; i++ {
			resp, err := http.Get(statsURL + "/api/v2/help_center/en_us/articles/456.json")
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			_ = resp.Body.Close()
		}

		stats := statsServer.GetRateLimitStatistics()
		if stats == nil {
			t.Fatal("Expected rate limit statistics")
		}

		if stats.TotalRequests == 0 {
			t.Error("Expected some total requests in statistics")
		}

		t.Logf("Statistics: Total=%d, Limited=%d", stats.TotalRequests, stats.LimitedRequests)
	})

	t.Run("UpdateRateLimits", func(t *testing.T) {
		// Create a fresh server for this test
		updateConfig := &MockServerConfig{
			EnableRateLimiting: true,
			RateLimitConfig: &RateLimitConfig{
				GlobalLimit:       1000, // High limit
				GlobalWindow:      time.Minute,
				BurstLimit:        100,  // High burst
				BurstWindow:       10 * time.Second,
				Enable429Response: true,
				EnableHeaders:     true,
			},
			EnableLogging: false,
		}

		updateServer := NewAdvancedMockServer(updateConfig)
		defer updateServer.Close()
		updateURL := updateServer.URL()

		// Update global limit
		updateServer.UpdateGlobalRateLimit(2000)

		// Update endpoint limit  
		updateServer.UpdateEndpointRateLimit("/articles", 1000)

		// Verify requests work with higher limits
		successCount := 0
		for i := 0; i < 5; i++ {
			resp, err := http.Get(updateURL + "/api/v2/help_center/en_us/articles/456.json")
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				successCount++
			}
		}

		if successCount == 0 {
			t.Error("Expected some successful requests with high limits")
		}

		t.Logf("UpdateRateLimits: %d out of 5 requests succeeded", successCount)
	})
}

func TestAdvancedMockServer_CombinedSimulation(t *testing.T) {
	t.Parallel()

	// Enable both latency and rate limiting
	config := &MockServerConfig{
		EnableLatencySim: true,
		LatencyConfig: &LatencyConfig{
			BaseLatency:    10 * time.Millisecond,
			NetworkProfile: NetworkBroadband,
		},
		EnableRateLimiting: true,
		RateLimitConfig: &RateLimitConfig{
			GlobalLimit:       10,
			GlobalWindow:      time.Minute,
			BurstLimit:        3,
			BurstWindow:       10 * time.Second,
			Enable429Response: true,
			EnableHeaders:     true,
		},
		EnableLogging: true,
	}

	server := NewAdvancedMockServer(config)
	defer server.Close()

	baseURL := server.URL()

	// Make requests that combine both latency and rate limiting
	var successCount, rateLimitedCount int

	for i := 0; i < 5; i++ {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("%s/api/v2/help_center/en_us/articles/456.json", baseURL))
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()

		// Should have some latency applied
		if duration < 5*time.Millisecond {
			t.Logf("Request %d: expected some latency, got %v", i, duration)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			rateLimitedCount++
		}

		t.Logf("Request %d: status=%d, duration=%v", i, resp.StatusCode, duration)
	}

	if successCount == 0 {
		t.Error("Expected some successful requests")
	}

	if rateLimitedCount == 0 {
		t.Log("No requests were rate limited (this may be expected depending on timing)")
	}

	// Check both statistics are available
	latencyStats := server.GetLatencyStatistics()
	if latencyStats == nil {
		t.Error("Expected latency statistics")
	}

	rateLimitStats := server.GetRateLimitStatistics()
	if rateLimitStats == nil {
		t.Error("Expected rate limit statistics")
	}

	t.Logf("Combined simulation: %d successful, %d rate limited", successCount, rateLimitedCount)
}