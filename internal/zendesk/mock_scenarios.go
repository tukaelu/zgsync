package zendesk

import (
	"net/http"
	"strings"
	"time"
)

// initializeDefaultScenarios sets up predefined behavior scenarios
func (sm *ScenarioManager) initializeDefaultScenarios() {
	sm.behaviors["Normal"] = &BehaviorSet{
		Name:            "Normal",
		ErrorRate:       0.0,
		LatencyMin:      5 * time.Millisecond,
		LatencyMax:      20 * time.Millisecond,
		FailurePatterns: make(map[string]ErrorPattern),
		CustomHandlers:  make(map[string]http.HandlerFunc),
	}

	sm.behaviors["HighLatency"] = &BehaviorSet{
		Name:            "HighLatency",
		ErrorRate:       0.0,
		LatencyMin:      200 * time.Millisecond,
		LatencyMax:      500 * time.Millisecond,
		FailurePatterns: make(map[string]ErrorPattern),
		CustomHandlers:  make(map[string]http.HandlerFunc),
	}

	sm.behaviors["Unreliable"] = &BehaviorSet{
		Name:       "Unreliable",
		ErrorRate:  0.3, // 30% error rate
		LatencyMin: 10 * time.Millisecond,
		LatencyMax: 100 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"random_500": {
				StatusCode: http.StatusInternalServerError,
				Response:   `{"error": "Internal Server Error", "description": "Simulated server error"}`,
				Condition: func(r *http.Request) bool {
					// Simple pseudo-random based on request path length
					return len(r.URL.Path)%3 == 0
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}

	sm.behaviors["RateLimited"] = &BehaviorSet{
		Name:       "RateLimited",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"rate_limit": {
				StatusCode: http.StatusTooManyRequests,
				Response:   `{"error": "Rate limit exceeded", "description": "API rate limit has been exceeded"}`,
				Condition: func(r *http.Request) bool {
					// Simulate rate limiting on rapid requests
					return strings.Contains(r.URL.Path, "articles") && r.Method == "POST"
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}

	sm.behaviors["AuthFailure"] = &BehaviorSet{
		Name:       "AuthFailure",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"auth_failure": {
				StatusCode: http.StatusUnauthorized,
				Response:   `{"error": "Unauthorized", "description": "Authentication credentials invalid"}`,
				Condition: func(r *http.Request) bool {
					// Always fail authentication in this scenario
					return true
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}

	sm.behaviors["PartialOutage"] = &BehaviorSet{
		Name:       "PartialOutage",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"articles_down": {
				StatusCode: http.StatusServiceUnavailable,
				Response:   `{"error": "Service Unavailable", "description": "Article service is temporarily unavailable"}`,
				Condition: func(r *http.Request) bool {
					// Only article endpoints are affected
					return strings.Contains(r.URL.Path, "articles")
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}

	sm.behaviors["SlowTranslations"] = &BehaviorSet{
		Name:            "SlowTranslations",
		ErrorRate:       0.0,
		LatencyMin:      5 * time.Millisecond,
		LatencyMax:      20 * time.Millisecond,
		FailurePatterns: make(map[string]ErrorPattern),
		CustomHandlers:  make(map[string]http.HandlerFunc),
	}

	// Add custom latency for translation endpoints in SlowTranslations scenario
	sm.behaviors["SlowTranslations"].FailurePatterns["slow_translations"] = ErrorPattern{
		StatusCode: 0, // 0 means don't return error, just apply latency
		Response:   "",
		Condition: func(r *http.Request) bool {
			if strings.Contains(r.URL.Path, "translations") {
				// Apply extra latency for translation endpoints
				time.Sleep(1 * time.Second)
			}
			return false // Don't actually return an error
		},
	}

	sm.behaviors["ValidationStrict"] = &BehaviorSet{
		Name:       "ValidationStrict",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"validation_error": {
				StatusCode: http.StatusBadRequest,
				Response:   `{"error": "Validation Error", "description": "Request validation failed", "details": [{"field": "title", "message": "Title is required"}]}`,
				Condition: func(r *http.Request) bool {
					// Simulate strict validation - reject POST requests without proper content-type
					return r.Method == "POST" && r.Header.Get("Content-Type") != "application/json"
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}

	sm.behaviors["DataCorruption"] = &BehaviorSet{
		Name:       "DataCorruption",
		ErrorRate:  0.0,
		LatencyMin: 5 * time.Millisecond,
		LatencyMax: 20 * time.Millisecond,
		FailurePatterns: map[string]ErrorPattern{
			"corrupted_response": {
				StatusCode: http.StatusOK,
				Response:   `{"article": {"id": "invalid", "title": null, "corrupted_field"`,
				Condition: func(r *http.Request) bool {
					// Return corrupted JSON for GET requests
					return r.Method == "GET" && strings.Contains(r.URL.Path, "articles")
				},
			},
		},
		CustomHandlers: make(map[string]http.HandlerFunc),
	}
}

// GetScenario returns the current active scenario
func (sm *ScenarioManager) GetScenario() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.activeScenario
}

// GetAvailableScenarios returns all available scenario names
func (sm *ScenarioManager) GetAvailableScenarios() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	scenarios := make([]string, 0, len(sm.behaviors))
	for name := range sm.behaviors {
		scenarios = append(scenarios, name)
	}
	return scenarios
}

// AddCustomScenario allows adding new behavior scenarios at runtime
func (sm *ScenarioManager) AddCustomScenario(name string, behavior *BehaviorSet) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.behaviors[name] = behavior
}

// RemoveScenario removes a scenario (cannot remove built-in scenarios)
func (sm *ScenarioManager) RemoveScenario(name string) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Protect built-in scenarios
	builtInScenarios := map[string]bool{
		"Normal":           true,
		"HighLatency":      true,
		"Unreliable":       true,
		"RateLimited":      true,
		"AuthFailure":      true,
		"PartialOutage":    true,
		"SlowTranslations": true,
		"ValidationStrict": true,
		"DataCorruption":   true,
	}

	if builtInScenarios[name] {
		return false
	}

	if _, exists := sm.behaviors[name]; !exists {
		return false
	}

	delete(sm.behaviors, name)

	// If the deleted scenario was active, switch to Normal
	if sm.activeScenario == name {
		sm.activeScenario = "Normal"
	}

	return true
}

// GetScenarioDetails returns detailed information about a scenario
func (sm *ScenarioManager) GetScenarioDetails(name string) *BehaviorSet {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	behavior, exists := sm.behaviors[name]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &BehaviorSet{
		Name:            behavior.Name,
		ErrorRate:       behavior.ErrorRate,
		LatencyMin:      behavior.LatencyMin,
		LatencyMax:      behavior.LatencyMax,
		FailurePatterns: behavior.FailurePatterns, // Note: shallow copy
		CustomHandlers:  behavior.CustomHandlers,  // Note: shallow copy
	}
}

// CreateTestScenario creates a scenario optimized for specific test cases
func (sm *ScenarioManager) CreateTestScenario(name string, config TestScenarioConfig) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	behavior := &BehaviorSet{
		Name:            name,
		ErrorRate:       config.ErrorRate,
		LatencyMin:      config.MinLatency,
		LatencyMax:      config.MaxLatency,
		FailurePatterns: make(map[string]ErrorPattern),
		CustomHandlers:  make(map[string]http.HandlerFunc),
	}

	// Add configured error patterns
	for patternName, pattern := range config.ErrorPatterns {
		behavior.FailurePatterns[patternName] = pattern
	}

	sm.behaviors[name] = behavior
}

// TestScenarioConfig provides configuration for test-specific scenarios
type TestScenarioConfig struct {
	ErrorRate     float64
	MinLatency    time.Duration
	MaxLatency    time.Duration
	ErrorPatterns map[string]ErrorPattern
}

// PredefinedTestScenarios provides commonly used test scenario configurations
var PredefinedTestScenarios = map[string]TestScenarioConfig{
	"FastAndReliable": {
		ErrorRate:     0.0,
		MinLatency:    1 * time.Millisecond,
		MaxLatency:    5 * time.Millisecond,
		ErrorPatterns: make(map[string]ErrorPattern),
	},
	"SlowButReliable": {
		ErrorRate:     0.0,
		MinLatency:    100 * time.Millisecond,
		MaxLatency:    200 * time.Millisecond,
		ErrorPatterns: make(map[string]ErrorPattern),
	},
	"NetworkIssues": {
		ErrorRate:  0.5,
		MinLatency: 50 * time.Millisecond,
		MaxLatency: 1000 * time.Millisecond,
		ErrorPatterns: map[string]ErrorPattern{
			"timeout": {
				StatusCode: http.StatusRequestTimeout,
				Response:   `{"error": "Request Timeout"}`,
				Condition: func(r *http.Request) bool {
					return len(r.URL.Path)%4 == 0
				},
			},
		},
	},
}
