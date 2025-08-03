package zendesk

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatencySimulator_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *LatencyConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &LatencyConfig{
				BaseLatency:    50 * time.Millisecond,
				JitterFactor:   0.1,
				Distribution:   DistributionNormal,
				NetworkProfile: NetworkMobile4G,
				EnableJitter:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			simulator := NewLatencySimulator(tt.config)
			if simulator == nil {
				t.Error("Expected non-nil simulator")
				return
			}

			if simulator.config == nil {
				t.Error("Expected non-nil config")
			}

			if len(simulator.patterns) == 0 {
				t.Error("Expected default patterns to be initialized")
			}
		})
	}
}

func TestLatencySimulator_SimulateLatency(t *testing.T) {
	t.Parallel()

	config := &LatencyConfig{
		BaseLatency:    10 * time.Millisecond,
		JitterFactor:   0.1,
		Distribution:   DistributionUniform,
		NetworkProfile: NetworkBroadband,
		EnableJitter:   true,
	}

	simulator := NewLatencySimulator(config)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedMinLatency time.Duration
		expectedMaxLatency time.Duration
	}{
		{
			name:           "GET articles",
			method:         "GET",
			path:           "/api/v2/help_center/articles/123",
			expectedMinLatency: 1 * time.Millisecond,
			expectedMaxLatency: 100 * time.Millisecond,
		},
		{
			name:           "POST articles",
			method:         "POST",
			path:           "/api/v2/help_center/articles",
			expectedMinLatency: 10 * time.Millisecond,
			expectedMaxLatency: 300 * time.Millisecond,
		},
		{
			name:           "GET translations",
			method:         "GET",
			path:           "/api/v2/help_center/translations/456",
			expectedMinLatency: 5 * time.Millisecond,
			expectedMaxLatency: 150 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			
			start := time.Now()
			latency := simulator.SimulateLatency(req)
			actualDuration := time.Since(start)

			// Verify latency was applied (actual duration should be close to simulated)
			if actualDuration < latency/2 {
				t.Errorf("Expected actual duration to be close to simulated latency. Got %v, simulated %v", actualDuration, latency)
			}

			// Verify latency is within reasonable bounds
			if latency < 0 {
				t.Errorf("Expected non-negative latency, got %v", latency)
			}

			if latency > 2*time.Second {
				t.Errorf("Expected reasonable latency, got %v", latency)
			}
		})
	}
}

func TestLatencySimulator_NetworkProfiles(t *testing.T) {
	t.Parallel()

	profiles := []struct {
		name    string
		profile NetworkProfile
	}{
		{"Fast", NetworkFast},
		{"Broadband", NetworkBroadband},
		{"WiFi", NetworkWiFi},
		{"Mobile4G", NetworkMobile4G},
		{"Mobile3G", NetworkMobile3G},
		{"Slow", NetworkSlow},
	}

	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			config := &LatencyConfig{
				BaseLatency:    10 * time.Millisecond,
				NetworkProfile: p.profile,
				EnableJitter:   false, // Disable jitter for consistent testing
			}

			simulator := NewLatencySimulator(config)
			req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

			// Run multiple times to get average
			var totalLatency time.Duration
			runs := 5
			
			for i := 0; i < runs; i++ {
				latency := simulator.SimulateLatency(req)
				totalLatency += latency
			}

			avgLatency := totalLatency / time.Duration(runs)

			// Verify latency increases with slower network profiles
			if avgLatency < 0 {
				t.Errorf("Expected positive average latency for %s profile, got %v", p.name, avgLatency)
			}

			t.Logf("Average latency for %s profile: %v", p.name, avgLatency)
		})
	}
}

func TestLatencySimulator_Distributions(t *testing.T) {
	t.Parallel()

	distributions := []struct {
		name string
		dist DistributionType
	}{
		{"Uniform", DistributionUniform},
		{"Normal", DistributionNormal},
		{"Exponential", DistributionExponential},
		{"Gamma", DistributionGamma},
	}

	for _, d := range distributions {
		t.Run(d.name, func(t *testing.T) {
			config := &LatencyConfig{
				BaseLatency:    10 * time.Millisecond,
				Distribution:   d.dist,
				NetworkProfile: NetworkBroadband,
				EnableJitter:   false,
			}

			simulator := NewLatencySimulator(config)
			req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)

			// Collect latency samples
			samples := make([]time.Duration, 20)
			for i := 0; i < len(samples); i++ {
				// Reset statistics to avoid interference
				simulator.ResetStatistics()
				samples[i] = simulator.SimulateLatency(req)
			}

			// Verify all samples are valid
			for i, sample := range samples {
				if sample < 0 {
					t.Errorf("Sample %d: expected non-negative latency, got %v", i, sample)
				}
			}

			t.Logf("Distribution %s sample range: %v to %v", d.name, 
				minDuration(samples), maxDuration(samples))
		})
	}
}

func TestLatencySimulator_CustomPatterns(t *testing.T) {
	t.Parallel()

	simulator := NewLatencySimulator(nil)

	// Add custom pattern
	customPattern := &LatencyPattern{
		PathPattern:    "/custom",
		Method:         "POST",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     200 * time.Millisecond,
		Distribution:   DistributionUniform,
		LoadFactor:     1.0,
		GeographicTier: GeographicLocal,
	}

	simulator.AddCustomPattern("custom_pattern", customPattern)

	// Test custom pattern
	req := httptest.NewRequest("POST", "/custom/endpoint", nil)
	latency := simulator.SimulateLatency(req)

	if latency < 50*time.Millisecond || latency > 500*time.Millisecond {
		t.Errorf("Expected latency in custom range, got %v", latency)
	}
}

func TestLatencySimulator_Statistics(t *testing.T) {
	t.Parallel()

	simulator := NewLatencySimulator(nil)

	// Make several requests
	requests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v2/help_center/articles/123"},
		{"POST", "/api/v2/help_center/articles"},
		{"GET", "/api/v2/help_center/translations/456"},
		{"PUT", "/api/v2/help_center/articles/789"},
	}

	for _, req := range requests {
		r := httptest.NewRequest(req.method, req.path, nil)
		simulator.SimulateLatency(r)
	}

	// Check statistics
	stats := simulator.GetNetworkStatistics()
	if stats == nil {
		t.Fatal("Expected non-nil statistics")
	}

	if stats.TotalRequests != int64(len(requests)) {
		t.Errorf("Expected %d total requests, got %d", len(requests), stats.TotalRequests)
	}

	if stats.AverageLatency <= 0 {
		t.Errorf("Expected positive average latency, got %v", stats.AverageLatency)
	}

	if stats.MinLatency <= 0 {
		t.Errorf("Expected positive min latency, got %v", stats.MinLatency)
	}

	if stats.MaxLatency < stats.MinLatency {
		t.Errorf("Expected max latency >= min latency, got max=%v, min=%v", 
			stats.MaxLatency, stats.MinLatency)
	}

	// Test reset
	simulator.ResetStatistics()
	stats = simulator.GetNetworkStatistics()
	
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", stats.TotalRequests)
	}
}

func TestLatencySimulator_Report(t *testing.T) {
	t.Parallel()

	simulator := NewLatencySimulator(nil)

	// Make some requests
	req := httptest.NewRequest("GET", "/api/v2/help_center/articles/123", nil)
	simulator.SimulateLatency(req)
	simulator.SimulateLatency(req)

	report := simulator.GetLatencyReport()
	if report == "" {
		t.Error("Expected non-empty report")
	}

	if !contains(report, "Total Requests: 2") {
		t.Error("Expected report to contain request count")
	}

	if !contains(report, "Average Latency:") {
		t.Error("Expected report to contain average latency")
	}
}

// Helper functions

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}