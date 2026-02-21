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
