package zendesk

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LatencySimulator provides realistic network latency simulation
type LatencySimulator struct {
	config       *LatencyConfig
	patterns     map[string]*LatencyPattern
	networkStats *NetworkStatistics
	mutex        sync.RWMutex
}

// LatencyConfig controls overall latency simulation behavior
type LatencyConfig struct {
	BaseLatency    time.Duration    // Base latency for all requests
	JitterFactor   float64          // Randomness factor (0.0-1.0)
	Distribution   DistributionType // Distribution type for latency
	NetworkProfile NetworkProfile   // Simulated network conditions
	EnableJitter   bool             // Enable/disable jitter
}

// LatencyPattern defines latency characteristics for specific endpoints
type LatencyPattern struct {
	PathPattern    string           // URL path pattern to match
	Method         string           // HTTP method
	MinLatency     time.Duration    // Minimum latency
	MaxLatency     time.Duration    // Maximum latency
	Distribution   DistributionType // Distribution type
	LoadFactor     float64          // Load-based latency multiplier
	GeographicTier GeographicTier   // Geographic distance simulation
}

// DistributionType defines how latency values are distributed
type DistributionType int

const (
	DistributionUniform DistributionType = iota
	DistributionNormal
	DistributionExponential
	DistributionGamma
)

// NetworkProfile simulates different network conditions
type NetworkProfile int

const (
	NetworkFast      NetworkProfile = iota // Fast broadband
	NetworkBroadband                       // Standard broadband
	NetworkWiFi                            // WiFi connection
	NetworkMobile4G                        // 4G mobile
	NetworkMobile3G                        // 3G mobile
	NetworkSlow                            // Slow connection
)

// GeographicTier simulates geographic distance effects
type GeographicTier int

const (
	GeographicLocal    GeographicTier = iota // Local/same region
	GeographicRegional                       // Same continent
	GeographicGlobal                         // Cross-continental
)

// NetworkStatistics tracks latency metrics
type NetworkStatistics struct {
	TotalRequests     int64
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	LatencyHistogram  map[time.Duration]int64
	RequestsByPattern map[string]int64
	mutex             sync.RWMutex
}

// NewLatencySimulator creates a new latency simulator
func NewLatencySimulator(config *LatencyConfig) *LatencySimulator {
	if config == nil {
		config = &LatencyConfig{
			BaseLatency:    10 * time.Millisecond,
			JitterFactor:   0.2,
			Distribution:   DistributionNormal,
			NetworkProfile: NetworkBroadband,
			EnableJitter:   true,
		}
	}

	simulator := &LatencySimulator{
		config:   config,
		patterns: make(map[string]*LatencyPattern),
		networkStats: &NetworkStatistics{
			LatencyHistogram:  make(map[time.Duration]int64),
			RequestsByPattern: make(map[string]int64),
		},
	}

	// Initialize default patterns
	simulator.initializeDefaultPatterns()

	return simulator
}

// initializeDefaultPatterns sets up realistic latency patterns for different endpoints
func (ls *LatencySimulator) initializeDefaultPatterns() {
	// Article operations - generally fast
	ls.patterns["articles_get"] = &LatencyPattern{
		PathPattern:    "/articles/",
		Method:         "GET",
		MinLatency:     5 * time.Millisecond,
		MaxLatency:     50 * time.Millisecond,
		Distribution:   DistributionNormal,
		LoadFactor:     1.0,
		GeographicTier: GeographicLocal,
	}

	// Article creation - slower due to processing
	ls.patterns["articles_post"] = &LatencyPattern{
		PathPattern:    "/articles",
		Method:         "POST",
		MinLatency:     50 * time.Millisecond,
		MaxLatency:     200 * time.Millisecond,
		Distribution:   DistributionGamma,
		LoadFactor:     1.5,
		GeographicTier: GeographicLocal,
	}

	// Article updates - moderate latency
	ls.patterns["articles_put"] = &LatencyPattern{
		PathPattern:    "/articles/",
		Method:         "PUT",
		MinLatency:     30 * time.Millisecond,
		MaxLatency:     150 * time.Millisecond,
		Distribution:   DistributionNormal,
		LoadFactor:     1.2,
		GeographicTier: GeographicLocal,
	}

	// Translation operations - can be slower due to content processing
	ls.patterns["translations_get"] = &LatencyPattern{
		PathPattern:    "/translations",
		Method:         "GET",
		MinLatency:     10 * time.Millisecond,
		MaxLatency:     80 * time.Millisecond,
		Distribution:   DistributionNormal,
		LoadFactor:     1.1,
		GeographicTier: GeographicRegional,
	}

	ls.patterns["translations_post"] = &LatencyPattern{
		PathPattern:    "/translations",
		Method:         "POST",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     500 * time.Millisecond,
		Distribution:   DistributionExponential,
		LoadFactor:     2.0,
		GeographicTier: GeographicRegional,
	}

	// Section operations - typically fast
	ls.patterns["sections_get"] = &LatencyPattern{
		PathPattern:    "/sections",
		Method:         "GET",
		MinLatency:     3 * time.Millisecond,
		MaxLatency:     25 * time.Millisecond,
		Distribution:   DistributionUniform,
		LoadFactor:     0.8,
		GeographicTier: GeographicLocal,
	}
}

// SimulateLatency calculates and applies realistic latency for a request
func (ls *LatencySimulator) SimulateLatency(r *http.Request) time.Duration {
	startTime := time.Now()

	// Find matching pattern
	pattern := ls.findMatchingPattern(r)

	// Calculate base latency
	baseLatency := ls.calculateBaseLatency(pattern)

	// Apply network profile effects
	networkLatency := ls.applyNetworkProfile(baseLatency)

	// Apply geographic effects
	geoLatency := ls.applyGeographicEffects(networkLatency, pattern.GeographicTier)

	// Apply load-based effects
	loadLatency := ls.applyLoadEffects(geoLatency, pattern.LoadFactor)

	// Apply jitter if enabled
	finalLatency := ls.applyJitter(loadLatency)

	// Record statistics
	ls.recordLatencyStats(pattern, finalLatency)

	// Apply the latency (sleep)
	time.Sleep(finalLatency)

	// Track actual processing time
	actualDuration := time.Since(startTime)
	ls.updateNetworkStats(actualDuration, pattern)

	return finalLatency
}

// findMatchingPattern finds the best matching latency pattern for a request
func (ls *LatencySimulator) findMatchingPattern(r *http.Request) *LatencyPattern {
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()

	path := r.URL.Path
	method := r.Method

	// Find exact matches first
	for _, pattern := range ls.patterns {
		if pattern.Method == method && strings.Contains(path, pattern.PathPattern) {
			return pattern
		}
	}

	// Return default pattern if no match found
	return &LatencyPattern{
		PathPattern:    "default",
		Method:         method,
		MinLatency:     ls.config.BaseLatency,
		MaxLatency:     ls.config.BaseLatency * 3,
		Distribution:   ls.config.Distribution,
		LoadFactor:     1.0,
		GeographicTier: GeographicLocal,
	}
}

// calculateBaseLatency calculates base latency using the specified distribution
func (ls *LatencySimulator) calculateBaseLatency(pattern *LatencyPattern) time.Duration {
	min := float64(pattern.MinLatency)
	max := float64(pattern.MaxLatency)

	var value float64

	switch pattern.Distribution {
	case DistributionUniform:
		value = min + rand.Float64()*(max-min)

	case DistributionNormal:
		mean := (min + max) / 2
		stddev := (max - min) / 6 // 99.7% within range
		value = rand.NormFloat64()*stddev + mean
		// Clamp to range
		if value < min {
			value = min
		}
		if value > max {
			value = max
		}

	case DistributionExponential:
		lambda := 1.0 / ((max - min) / 3) // Mean = 1/lambda
		value = min + rand.ExpFloat64()/lambda
		if value > max {
			value = max
		}

	case DistributionGamma:
		// Gamma distribution approximation using exponential
		// This is a simplified gamma distribution using multiple exponentials
		lambda := 2.0 / (max - min)
		value = min + rand.ExpFloat64()/lambda + rand.ExpFloat64()/lambda
		if value > max {
			value = max
		}

	default:
		value = min + rand.Float64()*(max-min)
	}

	return time.Duration(value)
}

// applyNetworkProfile applies network-specific latency characteristics
func (ls *LatencySimulator) applyNetworkProfile(baseLatency time.Duration) time.Duration {
	var multiplier float64
	var additionalLatency time.Duration

	switch ls.config.NetworkProfile {
	case NetworkFast:
		multiplier = 0.8
		additionalLatency = 1 * time.Millisecond

	case NetworkBroadband:
		multiplier = 1.0
		additionalLatency = 2 * time.Millisecond

	case NetworkWiFi:
		multiplier = 1.2
		additionalLatency = 5 * time.Millisecond

	case NetworkMobile4G:
		multiplier = 1.5
		additionalLatency = 20 * time.Millisecond

	case NetworkMobile3G:
		multiplier = 3.0
		additionalLatency = 100 * time.Millisecond

	case NetworkSlow:
		multiplier = 5.0
		additionalLatency = 500 * time.Millisecond

	default:
		multiplier = 1.0
		additionalLatency = 0
	}

	return time.Duration(float64(baseLatency)*multiplier) + additionalLatency
}

// applyGeographicEffects adds latency based on simulated geographic distance
func (ls *LatencySimulator) applyGeographicEffects(latency time.Duration, tier GeographicTier) time.Duration {
	var additionalLatency time.Duration

	switch tier {
	case GeographicLocal:
		additionalLatency = 0

	case GeographicRegional:
		additionalLatency = 10*time.Millisecond +
			time.Duration(rand.Float64()*float64(20*time.Millisecond))

	case GeographicGlobal:
		additionalLatency = 50*time.Millisecond +
			time.Duration(rand.Float64()*float64(100*time.Millisecond))
	}

	return latency + additionalLatency
}

// applyLoadEffects simulates server load-based latency increases
func (ls *LatencySimulator) applyLoadEffects(latency time.Duration, loadFactor float64) time.Duration {
	// Simulate varying server load (0.5 to 2.0)
	currentLoad := 0.5 + rand.Float64()*1.5

	// Apply load-based multiplier
	loadMultiplier := 1.0 + (currentLoad-1.0)*loadFactor*0.5

	return time.Duration(float64(latency) * loadMultiplier)
}

// applyJitter adds realistic network jitter if enabled
func (ls *LatencySimulator) applyJitter(latency time.Duration) time.Duration {
	if !ls.config.EnableJitter {
		return latency
	}

	jitterRange := float64(latency) * ls.config.JitterFactor
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange // -jitterRange to +jitterRange

	finalLatency := time.Duration(float64(latency) + jitter)

	// Ensure non-negative
	if finalLatency < 0 {
		finalLatency = 0
	}

	return finalLatency
}

// recordLatencyStats records latency statistics for analysis
func (ls *LatencySimulator) recordLatencyStats(pattern *LatencyPattern, latency time.Duration) {
	ls.networkStats.mutex.Lock()
	defer ls.networkStats.mutex.Unlock()

	// Update pattern-specific stats
	ls.networkStats.RequestsByPattern[pattern.PathPattern]++

	// Update histogram (bucketize to 5ms buckets)
	bucket := (latency / (5 * time.Millisecond)) * (5 * time.Millisecond)
	ls.networkStats.LatencyHistogram[bucket]++
}

// updateNetworkStats updates overall network statistics
func (ls *LatencySimulator) updateNetworkStats(actualLatency time.Duration, pattern *LatencyPattern) {
	ls.networkStats.mutex.Lock()
	defer ls.networkStats.mutex.Unlock()

	ls.networkStats.TotalRequests++

	// Update min/max
	if ls.networkStats.TotalRequests == 1 {
		ls.networkStats.MinLatency = actualLatency
		ls.networkStats.MaxLatency = actualLatency
		ls.networkStats.AverageLatency = actualLatency
	} else {
		if actualLatency < ls.networkStats.MinLatency {
			ls.networkStats.MinLatency = actualLatency
		}
		if actualLatency > ls.networkStats.MaxLatency {
			ls.networkStats.MaxLatency = actualLatency
		}

		// Update rolling average
		oldAvg := float64(ls.networkStats.AverageLatency)
		newAvg := oldAvg + (float64(actualLatency)-oldAvg)/float64(ls.networkStats.TotalRequests)
		ls.networkStats.AverageLatency = time.Duration(newAvg)
	}
}

// AddCustomPattern adds a custom latency pattern
func (ls *LatencySimulator) AddCustomPattern(name string, pattern *LatencyPattern) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.patterns[name] = pattern
}

// GetNetworkStatistics returns current network statistics
func (ls *LatencySimulator) GetNetworkStatistics() *NetworkStatistics {
	ls.networkStats.mutex.RLock()
	defer ls.networkStats.mutex.RUnlock()

	// Return a copy to prevent external modification
	stats := &NetworkStatistics{
		TotalRequests:     ls.networkStats.TotalRequests,
		AverageLatency:    ls.networkStats.AverageLatency,
		MinLatency:        ls.networkStats.MinLatency,
		MaxLatency:        ls.networkStats.MaxLatency,
		LatencyHistogram:  make(map[time.Duration]int64),
		RequestsByPattern: make(map[string]int64),
	}

	// Deep copy maps
	for k, v := range ls.networkStats.LatencyHistogram {
		stats.LatencyHistogram[k] = v
	}
	for k, v := range ls.networkStats.RequestsByPattern {
		stats.RequestsByPattern[k] = v
	}

	return stats
}

// ResetStatistics clears all collected statistics
func (ls *LatencySimulator) ResetStatistics() {
	ls.networkStats.mutex.Lock()
	defer ls.networkStats.mutex.Unlock()

	ls.networkStats.TotalRequests = 0
	ls.networkStats.AverageLatency = 0
	ls.networkStats.MinLatency = 0
	ls.networkStats.MaxLatency = 0
	ls.networkStats.LatencyHistogram = make(map[time.Duration]int64)
	ls.networkStats.RequestsByPattern = make(map[string]int64)
}

// SetNetworkProfile updates the network profile
func (ls *LatencySimulator) SetNetworkProfile(profile NetworkProfile) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.config.NetworkProfile = profile
}

// SetJitterEnabled enables or disables jitter
func (ls *LatencySimulator) SetJitterEnabled(enabled bool) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.config.EnableJitter = enabled
}

// GetLatencyReport generates a detailed latency report
func (ls *LatencySimulator) GetLatencyReport() string {
	stats := ls.GetNetworkStatistics()

	if stats.TotalRequests == 0 {
		return "No requests processed yet"
	}

	report := "Latency Simulation Report\n"
	report += "========================\n"
	report += fmt.Sprintf("Total Requests: %d\n", stats.TotalRequests)
	report += fmt.Sprintf("Average Latency: %v\n", stats.AverageLatency)
	report += fmt.Sprintf("Min Latency: %v\n", stats.MinLatency)
	report += fmt.Sprintf("Max Latency: %v\n", stats.MaxLatency)
	report += "\nRequests by Pattern:\n"

	for pattern, count := range stats.RequestsByPattern {
		report += fmt.Sprintf("  %s: %d requests\n", pattern, count)
	}

	report += "\nLatency Distribution (5ms buckets):\n"
	for bucket, count := range stats.LatencyHistogram {
		if count > 0 {
			report += fmt.Sprintf("  %v: %d requests\n", bucket, count)
		}
	}

	return report
}
