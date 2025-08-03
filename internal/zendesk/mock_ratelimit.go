package zendesk

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter provides realistic API rate limiting simulation
type RateLimiter struct {
	config           *RateLimitConfig
	buckets          map[string]*TokenBucket
	requestHistory   *RequestHistory
	rateLimitStats   *RateLimitStatistics
	mutex            sync.RWMutex
}

// RateLimitConfig controls rate limiting behavior
type RateLimitConfig struct {
	GlobalLimit       int           // Global requests per window
	GlobalWindow      time.Duration // Global rate limit window
	BurstLimit        int           // Maximum burst requests
	BurstWindow       time.Duration // Burst window duration
	PerEndpointLimits map[string]int // Per-endpoint limits
	Enable429Response bool          // Return 429 when limit exceeded
	EnableHeaders     bool          // Add rate limit headers
	GracePeriod       time.Duration // Grace period before enforcement
}

// TokenBucket implements token bucket algorithm for rate limiting
type TokenBucket struct {
	Capacity       int           // Maximum tokens
	Tokens         float64       // Current tokens
	RefillRate     float64       // Tokens per second
	LastRefill     time.Time     // Last refill time
	WindowStart    time.Time     // Current window start
	WindowRequests int           // Requests in current window
	WindowDuration time.Duration // Window duration
	mutex          sync.Mutex
}

// RequestHistory tracks recent requests for rate limiting
type RequestHistory struct {
	requests map[string][]time.Time // Key: client/endpoint, Value: request times
	mutex    sync.RWMutex
}

// RateLimitStatistics tracks rate limiting metrics
type RateLimitStatistics struct {
	TotalRequests      int64
	LimitedRequests    int64
	BurstRequests      int64
	AverageRate        float64
	PeakRate           float64
	WindowViolations   int64
	EndpointStats      map[string]*EndpointStats
	mutex              sync.RWMutex
}

// EndpointStats tracks statistics per endpoint
type EndpointStats struct {
	Requests        int64
	LimitedRequests int64
	LastLimitTime   time.Time
	CurrentRate     float64
}

// RateLimitResult represents the result of rate limit checking
type RateLimitResult struct {
	Allowed        bool
	Remaining      int
	ResetTime      time.Time
	RetryAfter     time.Duration
	LimitType      string // "global", "endpoint", "burst"
	CurrentRate    float64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = &RateLimitConfig{
			GlobalLimit:       100,                           // 100 requests per minute
			GlobalWindow:      time.Minute,
			BurstLimit:        20,                            // 20 requests per 10 seconds
			BurstWindow:       10 * time.Second,
			PerEndpointLimits: make(map[string]int),
			Enable429Response: true,
			EnableHeaders:     true,
			GracePeriod:       5 * time.Second,
		}

		// Default per-endpoint limits (realistic Zendesk API limits)
		config.PerEndpointLimits["/articles"] = 200  // Higher limit for reads
		config.PerEndpointLimits["/translations"] = 100
		config.PerEndpointLimits["/sections"] = 300
	}

	limiter := &RateLimiter{
		config:  config,
		buckets: make(map[string]*TokenBucket),
		requestHistory: &RequestHistory{
			requests: make(map[string][]time.Time),
		},
		rateLimitStats: &RateLimitStatistics{
			EndpointStats: make(map[string]*EndpointStats),
		},
	}

	// Initialize token buckets
	limiter.initializeTokenBuckets()

	return limiter
}

// initializeTokenBuckets sets up token buckets for different limits
func (rl *RateLimiter) initializeTokenBuckets() {
	now := time.Now()

	// Global rate limit bucket
	rl.buckets["global"] = &TokenBucket{
		Capacity:       rl.config.GlobalLimit,
		Tokens:         float64(rl.config.GlobalLimit),
		RefillRate:     float64(rl.config.GlobalLimit) / rl.config.GlobalWindow.Seconds(),
		LastRefill:     now,
		WindowStart:    now,
		WindowDuration: rl.config.GlobalWindow,
	}

	// Burst limit bucket
	rl.buckets["burst"] = &TokenBucket{
		Capacity:       rl.config.BurstLimit,
		Tokens:         float64(rl.config.BurstLimit),
		RefillRate:     float64(rl.config.BurstLimit) / rl.config.BurstWindow.Seconds(),
		LastRefill:     now,
		WindowStart:    now,
		WindowDuration: rl.config.BurstWindow,
	}

	// Per-endpoint buckets
	for endpoint, limit := range rl.config.PerEndpointLimits {
		rl.buckets[endpoint] = &TokenBucket{
			Capacity:       limit,
			Tokens:         float64(limit),
			RefillRate:     float64(limit) / rl.config.GlobalWindow.Seconds(),
			LastRefill:     now,
			WindowStart:    now,
			WindowDuration: rl.config.GlobalWindow,
		}
	}
}

// CheckRateLimit checks if a request should be rate limited
func (rl *RateLimiter) CheckRateLimit(r *http.Request) *RateLimitResult {
	now := time.Now()
	endpoint := rl.getEndpointKey(r)
	clientKey := rl.getClientKey(r)

	// Record request for statistics
	rl.recordRequest(endpoint, clientKey, now)

	// Check grace period
	if rl.isInGracePeriod(now) {
		return &RateLimitResult{
			Allowed:     true,
			Remaining:   rl.config.GlobalLimit,
			ResetTime:   now.Add(rl.config.GlobalWindow),
			CurrentRate: rl.getCurrentRate(endpoint),
		}
	}

	// Check burst limit first (shortest window)
	if result := rl.checkBurstLimit(now); !result.Allowed {
		result.LimitType = "burst"
		return result
	}

	// Check global limit
	if result := rl.checkGlobalLimit(now); !result.Allowed {
		result.LimitType = "global"
		return result
	}

	// Check endpoint-specific limit
	if result := rl.checkEndpointLimit(endpoint, now); !result.Allowed {
		result.LimitType = endpoint
		return result
	}

	// Request is allowed
	return &RateLimitResult{
		Allowed:     true,
		Remaining:   rl.getRemainingTokens("global"),
		ResetTime:   rl.getResetTime("global"),
		CurrentRate: rl.getCurrentRate(endpoint),
	}
}

// checkBurstLimit checks burst rate limit
func (rl *RateLimiter) checkBurstLimit(now time.Time) *RateLimitResult {
	bucket := rl.buckets["burst"]
	return rl.checkTokenBucket(bucket, now, "burst")
}

// checkGlobalLimit checks global rate limit
func (rl *RateLimiter) checkGlobalLimit(now time.Time) *RateLimitResult {
	bucket := rl.buckets["global"]
	return rl.checkTokenBucket(bucket, now, "global")
}

// checkEndpointLimit checks endpoint-specific rate limit
func (rl *RateLimiter) checkEndpointLimit(endpoint string, now time.Time) *RateLimitResult {
	bucket, exists := rl.buckets[endpoint]
	if !exists {
		// No specific limit for this endpoint
		return &RateLimitResult{Allowed: true}
	}

	return rl.checkTokenBucket(bucket, now, endpoint)
}

// checkTokenBucket checks if tokens are available in a bucket
func (rl *RateLimiter) checkTokenBucket(bucket *TokenBucket, now time.Time, bucketType string) *RateLimitResult {
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	// Refill tokens based on elapsed time
	rl.refillTokens(bucket, now)

	if bucket.Tokens >= 1.0 {
		// Consume one token
		bucket.Tokens -= 1.0
		bucket.WindowRequests++

		return &RateLimitResult{
			Allowed:   true,
			Remaining: int(bucket.Tokens),
			ResetTime: bucket.WindowStart.Add(bucket.WindowDuration),
		}
	}

	// Rate limit exceeded
	rl.recordRateLimitViolation(bucketType)

	retryAfter := rl.calculateRetryAfter(bucket)
	
	return &RateLimitResult{
		Allowed:    false,
		Remaining:  0,
		ResetTime:  bucket.WindowStart.Add(bucket.WindowDuration),
		RetryAfter: retryAfter,
	}
}

// refillTokens refills tokens in a bucket based on elapsed time
func (rl *RateLimiter) refillTokens(bucket *TokenBucket, now time.Time) {
	// Check if we need to reset the window
	if now.Sub(bucket.WindowStart) >= bucket.WindowDuration {
		bucket.WindowStart = now
		bucket.WindowRequests = 0
		bucket.Tokens = float64(bucket.Capacity)
		bucket.LastRefill = now
		return
	}

	// Refill based on elapsed time
	elapsed := now.Sub(bucket.LastRefill)
	tokensToAdd := bucket.RefillRate * elapsed.Seconds()
	
	bucket.Tokens = math.Min(float64(bucket.Capacity), bucket.Tokens+tokensToAdd)
	bucket.LastRefill = now
}

// calculateRetryAfter calculates when the client should retry
func (rl *RateLimiter) calculateRetryAfter(bucket *TokenBucket) time.Duration {
	// Time until next token is available
	if bucket.RefillRate > 0 {
		timeForOneToken := time.Duration(1.0/bucket.RefillRate) * time.Second
		return timeForOneToken
	}

	// Fallback to window reset
	return time.Until(bucket.WindowStart.Add(bucket.WindowDuration))
}

// ApplyRateLimit applies rate limiting to a response
func (rl *RateLimiter) ApplyRateLimit(w http.ResponseWriter, r *http.Request) bool {
	result := rl.CheckRateLimit(r)

	// Add rate limit headers if enabled
	if rl.config.EnableHeaders {
		rl.addRateLimitHeaders(w, result)
	}

	if !result.Allowed && rl.config.Enable429Response {
		// Return 429 Too Many Requests
		w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
		w.WriteHeader(http.StatusTooManyRequests)
		
		errorResponse := fmt.Sprintf(`{
			"error": "Rate limit exceeded",
			"description": "API rate limit exceeded for %s",
			"retry_after": %d,
			"limit_type": "%s"
		}`, result.LimitType, int(result.RetryAfter.Seconds()), result.LimitType)
		
		_, _ = w.Write([]byte(errorResponse))
		return true // Request was handled (rate limited)
	}

	return false // Request should proceed
}

// addRateLimitHeaders adds standard rate limit headers to the response
func (rl *RateLimiter) addRateLimitHeaders(w http.ResponseWriter, result *RateLimitResult) {
	w.Header().Set("X-Rate-Limit-Limit", strconv.Itoa(rl.config.GlobalLimit))
	w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-Rate-Limit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	
	if !result.Allowed {
		w.Header().Set("X-Rate-Limit-Type", result.LimitType)
	}
}

// Helper methods

func (rl *RateLimiter) getEndpointKey(r *http.Request) string {
	path := r.URL.Path
	
	// Normalize path to match configured endpoints
	for endpoint := range rl.config.PerEndpointLimits {
		if contains(path, endpoint) {
			return endpoint
		}
	}
	
	return "default"
}

func (rl *RateLimiter) getClientKey(r *http.Request) string {
	// In a real implementation, this might use IP, API key, user ID, etc.
	// For testing, we'll use a simple approach
	return r.RemoteAddr
}

func (rl *RateLimiter) isInGracePeriod(now time.Time) bool {
	// Simple grace period implementation
	// In a real system, this might be based on server startup time
	return false
}

func (rl *RateLimiter) recordRequest(endpoint, clientKey string, now time.Time) {
	rl.rateLimitStats.mutex.Lock()
	defer rl.rateLimitStats.mutex.Unlock()

	rl.rateLimitStats.TotalRequests++

	// Update endpoint stats
	if rl.rateLimitStats.EndpointStats[endpoint] == nil {
		rl.rateLimitStats.EndpointStats[endpoint] = &EndpointStats{}
	}
	
	stats := rl.rateLimitStats.EndpointStats[endpoint]
	stats.Requests++
	
	// Calculate current rate (requests per second over last minute)
	rl.updateCurrentRate(endpoint, now)
}

func (rl *RateLimiter) updateCurrentRate(endpoint string, now time.Time) {
	rl.requestHistory.mutex.Lock()
	defer rl.requestHistory.mutex.Unlock()

	key := endpoint
	if rl.requestHistory.requests[key] == nil {
		rl.requestHistory.requests[key] = make([]time.Time, 0)
	}

	// Add current request
	rl.requestHistory.requests[key] = append(rl.requestHistory.requests[key], now)

	// Remove requests older than 1 minute
	cutoff := now.Add(-time.Minute)
	requests := rl.requestHistory.requests[key]
	validRequests := make([]time.Time, 0)
	
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	
	rl.requestHistory.requests[key] = validRequests

	// Calculate rate (requests per second)
	rate := float64(len(validRequests)) / 60.0

	// Update statistics
	if stats := rl.rateLimitStats.EndpointStats[endpoint]; stats != nil {
		stats.CurrentRate = rate
		if rate > rl.rateLimitStats.PeakRate {
			rl.rateLimitStats.PeakRate = rate
		}
	}
}

func (rl *RateLimiter) recordRateLimitViolation(limitType string) {
	rl.rateLimitStats.mutex.Lock()
	defer rl.rateLimitStats.mutex.Unlock()

	rl.rateLimitStats.LimitedRequests++
	
	if limitType == "burst" {
		rl.rateLimitStats.BurstRequests++
	}
	
	rl.rateLimitStats.WindowViolations++
}

func (rl *RateLimiter) getRemainingTokens(bucketType string) int {
	bucket, exists := rl.buckets[bucketType]
	if !exists {
		return 0
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	return int(bucket.Tokens)
}

func (rl *RateLimiter) getResetTime(bucketType string) time.Time {
	bucket, exists := rl.buckets[bucketType]
	if !exists {
		return time.Now()
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	return bucket.WindowStart.Add(bucket.WindowDuration)
}

func (rl *RateLimiter) getCurrentRate(endpoint string) float64 {
	rl.rateLimitStats.mutex.RLock()
	defer rl.rateLimitStats.mutex.RUnlock()

	if stats := rl.rateLimitStats.EndpointStats[endpoint]; stats != nil {
		return stats.CurrentRate
	}
	
	return 0.0
}

// Public API methods

// GetStatistics returns current rate limiting statistics
func (rl *RateLimiter) GetStatistics() *RateLimitStatistics {
	rl.rateLimitStats.mutex.RLock()
	defer rl.rateLimitStats.mutex.RUnlock()

	// Return a copy
	stats := &RateLimitStatistics{
		TotalRequests:    rl.rateLimitStats.TotalRequests,
		LimitedRequests:  rl.rateLimitStats.LimitedRequests,
		BurstRequests:    rl.rateLimitStats.BurstRequests,
		AverageRate:      rl.rateLimitStats.AverageRate,
		PeakRate:         rl.rateLimitStats.PeakRate,
		WindowViolations: rl.rateLimitStats.WindowViolations,
		EndpointStats:    make(map[string]*EndpointStats),
	}

	// Deep copy endpoint stats
	for k, v := range rl.rateLimitStats.EndpointStats {
		stats.EndpointStats[k] = &EndpointStats{
			Requests:        v.Requests,
			LimitedRequests: v.LimitedRequests,
			LastLimitTime:   v.LastLimitTime,
			CurrentRate:     v.CurrentRate,
		}
	}

	return stats
}

// ResetStatistics clears all rate limiting statistics
func (rl *RateLimiter) ResetStatistics() {
	rl.rateLimitStats.mutex.Lock()
	defer rl.rateLimitStats.mutex.Unlock()

	rl.rateLimitStats.TotalRequests = 0
	rl.rateLimitStats.LimitedRequests = 0
	rl.rateLimitStats.BurstRequests = 0
	rl.rateLimitStats.AverageRate = 0
	rl.rateLimitStats.PeakRate = 0
	rl.rateLimitStats.WindowViolations = 0
	rl.rateLimitStats.EndpointStats = make(map[string]*EndpointStats)

	// Clear request history
	rl.requestHistory.mutex.Lock()
	rl.requestHistory.requests = make(map[string][]time.Time)
	rl.requestHistory.mutex.Unlock()
}

// UpdateGlobalLimit updates the global rate limit
func (rl *RateLimiter) UpdateGlobalLimit(limit int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.config.GlobalLimit = limit
	if bucket := rl.buckets["global"]; bucket != nil {
		bucket.mutex.Lock()
		bucket.Capacity = limit
		bucket.RefillRate = float64(limit) / rl.config.GlobalWindow.Seconds()
		bucket.mutex.Unlock()
	}
}

// UpdateEndpointLimit updates the limit for a specific endpoint
func (rl *RateLimiter) UpdateEndpointLimit(endpoint string, limit int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if rl.config.PerEndpointLimits == nil {
		rl.config.PerEndpointLimits = make(map[string]int)
	}
	rl.config.PerEndpointLimits[endpoint] = limit
	
	// Create or update bucket
	if rl.buckets[endpoint] == nil {
		rl.buckets[endpoint] = &TokenBucket{
			WindowStart:    time.Now(),
			WindowDuration: rl.config.GlobalWindow,
		}
	}
	
	bucket := rl.buckets[endpoint]
	bucket.mutex.Lock()
	bucket.Capacity = limit
	bucket.RefillRate = float64(limit) / rl.config.GlobalWindow.Seconds()
	bucket.mutex.Unlock()
}

// GetRateLimitReport generates a detailed rate limiting report
func (rl *RateLimiter) GetRateLimitReport() string {
	stats := rl.GetStatistics()
	
	report := "Rate Limiting Report\n"
	report += "====================\n"
	report += fmt.Sprintf("Total Requests: %d\n", stats.TotalRequests)
	report += fmt.Sprintf("Limited Requests: %d (%.2f%%)\n", stats.LimitedRequests, 
		float64(stats.LimitedRequests)/float64(stats.TotalRequests)*100)
	report += fmt.Sprintf("Burst Violations: %d\n", stats.BurstRequests)
	report += fmt.Sprintf("Peak Rate: %.2f req/s\n", stats.PeakRate)
	report += fmt.Sprintf("Window Violations: %d\n", stats.WindowViolations)
	
	report += "\nEndpoint Statistics:\n"
	for endpoint, endpointStats := range stats.EndpointStats {
		limitedPct := float64(0)
		if endpointStats.Requests > 0 {
			limitedPct = float64(endpointStats.LimitedRequests) / float64(endpointStats.Requests) * 100
		}
		
		report += fmt.Sprintf("  %s:\n", endpoint)
		report += fmt.Sprintf("    Requests: %d\n", endpointStats.Requests)
		report += fmt.Sprintf("    Limited: %d (%.2f%%)\n", endpointStats.LimitedRequests, limitedPct)
		report += fmt.Sprintf("    Current Rate: %.2f req/s\n", endpointStats.CurrentRate)
	}
	
	return report
}

// Utility function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s[0:len(substr)] == substr || 
		    s[len(s)-len(substr):] == substr ||
		    indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}