package zendesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ZendeskErrorResponse represents a realistic Zendesk API error response
type ZendeskErrorResponse struct {
	Error       string                 `json:"error"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ValidationError represents field validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorSimulator provides realistic error behavior simulation
type ErrorSimulator struct {
	scenarios map[string]*ErrorScenario
}

// ErrorScenario defines a specific error simulation scenario
type ErrorScenario struct {
	Name         string
	Probability  float64 // 0.0 to 1.0
	Errors       []ErrorDefinition
	BackoffDelay time.Duration
}

// ErrorDefinition defines a specific error type
type ErrorDefinition struct {
	StatusCode  int
	ErrorType   string
	Description string
	Condition   func(*http.Request) bool
	Details     map[string]interface{}
}

// NewErrorSimulator creates a new error simulator with predefined scenarios
func NewErrorSimulator() *ErrorSimulator {
	es := &ErrorSimulator{
		scenarios: make(map[string]*ErrorScenario),
	}
	es.initializePredefinedScenarios()
	return es
}

// initializePredefinedScenarios sets up realistic error scenarios
func (es *ErrorSimulator) initializePredefinedScenarios() {
	// Authentication and authorization errors
	es.scenarios["AuthenticationFailures"] = &ErrorScenario{
		Name:        "AuthenticationFailures",
		Probability: 0.1,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusUnauthorized,
				ErrorType:   "Unauthorized",
				Description: "Authentication credentials invalid",
				Condition: func(r *http.Request) bool {
					auth := r.Header.Get("Authorization")
					return auth == "" || !strings.HasPrefix(auth, "Basic ")
				},
			},
			{
				StatusCode:  http.StatusForbidden,
				ErrorType:   "Forbidden",
				Description: "Access denied. User does not have permission to perform this action",
				Condition: func(r *http.Request) bool {
					return r.Method == "DELETE" || (r.Method == "PUT" && strings.Contains(r.URL.Path, "articles"))
				},
			},
		},
	}

	// Rate limiting errors
	es.scenarios["RateLimiting"] = &ErrorScenario{
		Name:         "RateLimiting",
		Probability:  0.05,
		BackoffDelay: 30 * time.Second,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusTooManyRequests,
				ErrorType:   "TooManyRequests",
				Description: "API rate limit exceeded. Please try again later",
				Condition: func(r *http.Request) bool {
					return r.Method == "POST" || r.Method == "PUT"
				},
				Details: map[string]interface{}{
					"retry_after": 30,
					"rate_limit": map[string]interface{}{
						"limit":     100,
						"remaining": 0,
						"reset":     time.Now().Add(30 * time.Second).Unix(),
					},
				},
			},
		},
	}

	// Validation errors
	es.scenarios["ValidationErrors"] = &ErrorScenario{
		Name:        "ValidationErrors",
		Probability: 0.15,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusBadRequest,
				ErrorType:   "ValidationError",
				Description: "Request validation failed",
				Condition: func(r *http.Request) bool {
					return r.Method == "POST" || r.Method == "PUT"
				},
				Details: map[string]interface{}{
					"errors": []ValidationError{
						{
							Field:   "title",
							Code:    "required",
							Message: "Title is required",
						},
						{
							Field:   "title",
							Code:    "too_long",
							Message: "Title must be less than 255 characters",
						},
					},
				},
			},
			{
				StatusCode:  http.StatusUnprocessableEntity,
				ErrorType:   "UnprocessableEntity",
				Description: "The request could not be processed due to invalid data",
				Condition: func(r *http.Request) bool {
					return strings.Contains(r.URL.Path, "translations") && r.Method == "POST"
				},
				Details: map[string]interface{}{
					"errors": []ValidationError{
						{
							Field:   "locale",
							Code:    "invalid",
							Message: "Locale 'invalid_locale' is not supported",
						},
					},
				},
			},
		},
	}

	// Resource not found errors
	es.scenarios["ResourceNotFound"] = &ErrorScenario{
		Name:        "ResourceNotFound",
		Probability: 0.2,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusNotFound,
				ErrorType:   "RecordNotFound",
				Description: "The requested resource could not be found",
				Condition: func(r *http.Request) bool {
					return r.Method == "GET" && strings.Contains(r.URL.Path, "/999")
				},
			},
			{
				StatusCode:  http.StatusNotFound,
				ErrorType:   "SectionNotFound",
				Description: "Section not found",
				Condition: func(r *http.Request) bool {
					return strings.Contains(r.URL.Path, "sections/999")
				},
			},
		},
	}

	// Server errors
	es.scenarios["ServerErrors"] = &ErrorScenario{
		Name:        "ServerErrors",
		Probability: 0.05,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusInternalServerError,
				ErrorType:   "InternalServerError",
				Description: "An internal server error occurred",
				Condition: func(r *http.Request) bool {
					return len(r.URL.Path)%7 == 0 // Simple pseudo-random
				},
				Details: map[string]interface{}{
					"incident_id": "INC-" + fmt.Sprintf("%d", time.Now().Unix()),
				},
			},
			{
				StatusCode:  http.StatusBadGateway,
				ErrorType:   "BadGateway",
				Description: "Bad gateway - upstream service unavailable",
				Condition: func(r *http.Request) bool {
					return strings.Contains(r.URL.Path, "translations") && len(r.URL.Path)%11 == 0
				},
			},
			{
				StatusCode:  http.StatusServiceUnavailable,
				ErrorType:   "ServiceUnavailable",
				Description: "Service temporarily unavailable due to maintenance",
				Condition: func(r *http.Request) bool {
					return len(r.URL.Path)%13 == 0
				},
				Details: map[string]interface{}{
					"maintenance_window": map[string]interface{}{
						"start": time.Now().Format(time.RFC3339),
						"end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
					},
				},
			},
		},
	}

	// Network and timeout errors
	es.scenarios["NetworkErrors"] = &ErrorScenario{
		Name:        "NetworkErrors",
		Probability: 0.03,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusRequestTimeout,
				ErrorType:   "RequestTimeout",
				Description: "Request timed out",
				Condition: func(r *http.Request) bool {
					return len(r.URL.Path)%17 == 0
				},
			},
			{
				StatusCode:  http.StatusGatewayTimeout,
				ErrorType:   "GatewayTimeout",
				Description: "Gateway timeout - upstream service did not respond",
				Condition: func(r *http.Request) bool {
					return len(r.URL.Path)%19 == 0
				},
			},
		},
	}

	// Content-related errors
	es.scenarios["ContentErrors"] = &ErrorScenario{
		Name:        "ContentErrors",
		Probability: 0.08,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusBadRequest,
				ErrorType:   "InvalidContent",
				Description: "Content contains invalid or unsafe HTML",
				Condition: func(r *http.Request) bool {
					return r.Method == "POST" && strings.Contains(r.URL.Path, "articles")
				},
				Details: map[string]interface{}{
					"invalid_tags": []string{"<script>", "<iframe>", "<object>"},
				},
			},
			{
				StatusCode:  http.StatusRequestEntityTooLarge,
				ErrorType:   "PayloadTooLarge",
				Description: "Request payload is too large",
				Condition: func(r *http.Request) bool {
					return r.ContentLength > 0 && r.ContentLength > 1024*1024 // 1MB
				},
				Details: map[string]interface{}{
					"max_size": "1MB",
					"limit":    1024 * 1024,
				},
			},
		},
	}

	// Conflict errors
	es.scenarios["ConflictErrors"] = &ErrorScenario{
		Name:        "ConflictErrors",
		Probability: 0.1,
		Errors: []ErrorDefinition{
			{
				StatusCode:  http.StatusConflict,
				ErrorType:   "Conflict",
				Description: "The resource has been modified by another user",
				Condition: func(r *http.Request) bool {
					return r.Method == "PUT" && len(r.URL.Path)%23 == 0
				},
				Details: map[string]interface{}{
					"version_conflict": map[string]interface{}{
						"expected": 1,
						"actual":   2,
					},
				},
			},
			{
				StatusCode:  http.StatusConflict,
				ErrorType:   "DuplicateResource",
				Description: "A resource with this identifier already exists",
				Condition: func(r *http.Request) bool {
					return r.Method == "POST" && strings.Contains(r.URL.Path, "translations")
				},
			},
		},
	}
}

// SimulateError checks if an error should be simulated for the given request
func (es *ErrorSimulator) SimulateError(r *http.Request, scenarioName string) *ErrorSimulation {
	scenario, exists := es.scenarios[scenarioName]
	if !exists {
		return nil
	}

	// Check each error definition in the scenario
	for _, errorDef := range scenario.Errors {
		if errorDef.Condition != nil && errorDef.Condition(r) {
			return &ErrorSimulation{
				StatusCode:   errorDef.StatusCode,
				Response:     es.formatErrorResponse(errorDef),
				BackoffDelay: scenario.BackoffDelay,
			}
		}
	}

	return nil
}

// ErrorSimulation represents the result of error simulation
type ErrorSimulation struct {
	StatusCode   int
	Response     string
	BackoffDelay time.Duration
}

// formatErrorResponse creates a properly formatted Zendesk-style error response
func (es *ErrorSimulator) formatErrorResponse(errorDef ErrorDefinition) string {
	response := ZendeskErrorResponse{
		Error:       errorDef.ErrorType,
		Description: errorDef.Description,
		Details:     errorDef.Details,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		// Fallback to simple error format
		return fmt.Sprintf(`{"error": "%s", "description": "%s"}`, errorDef.ErrorType, errorDef.Description)
	}

	return string(jsonBytes)
}

// GetAvailableScenarios returns all available error scenarios
func (es *ErrorSimulator) GetAvailableScenarios() []string {
	scenarios := make([]string, 0, len(es.scenarios))
	for name := range es.scenarios {
		scenarios = append(scenarios, name)
	}
	return scenarios
}

// AddCustomScenario allows adding custom error scenarios
func (es *ErrorSimulator) AddCustomScenario(name string, scenario *ErrorScenario) {
	es.scenarios[name] = scenario
}

// GetScenarioDetails returns details about a specific scenario
func (es *ErrorSimulator) GetScenarioDetails(name string) *ErrorScenario {
	return es.scenarios[name]
}

// CreateCompositeScenario creates a scenario that combines multiple error types
func (es *ErrorSimulator) CreateCompositeScenario(name string, scenarios []string, probability float64) {
	composite := &ErrorScenario{
		Name:        name,
		Probability: probability,
		Errors:      make([]ErrorDefinition, 0),
	}

	// Combine errors from multiple scenarios
	for _, scenarioName := range scenarios {
		if scenario, exists := es.scenarios[scenarioName]; exists {
			composite.Errors = append(composite.Errors, scenario.Errors...)
		}
	}

	es.scenarios[name] = composite
}

// ErrorDistribution represents error occurrence statistics
type ErrorDistribution struct {
	ScenarioName string
	TotalChecks  int
	ErrorCount   int
	ErrorRate    float64
	LastError    time.Time
}

// ErrorTracker tracks error simulation statistics
type ErrorTracker struct {
	distributions map[string]*ErrorDistribution
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		distributions: make(map[string]*ErrorDistribution),
	}
}

// RecordCheck records an error simulation check
func (et *ErrorTracker) RecordCheck(scenarioName string, errorOccurred bool) {
	if et.distributions[scenarioName] == nil {
		et.distributions[scenarioName] = &ErrorDistribution{
			ScenarioName: scenarioName,
		}
	}

	dist := et.distributions[scenarioName]
	dist.TotalChecks++

	if errorOccurred {
		dist.ErrorCount++
		dist.LastError = time.Now()
	}

	if dist.TotalChecks > 0 {
		dist.ErrorRate = float64(dist.ErrorCount) / float64(dist.TotalChecks)
	}
}

// GetDistributions returns all error distribution statistics
func (et *ErrorTracker) GetDistributions() map[string]*ErrorDistribution {
	result := make(map[string]*ErrorDistribution)
	for name, dist := range et.distributions {
		// Return a copy to prevent external modification
		result[name] = &ErrorDistribution{
			ScenarioName: dist.ScenarioName,
			TotalChecks:  dist.TotalChecks,
			ErrorCount:   dist.ErrorCount,
			ErrorRate:    dist.ErrorRate,
			LastError:    dist.LastError,
		}
	}
	return result
}

// Reset clears all error statistics
func (et *ErrorTracker) Reset() {
	et.distributions = make(map[string]*ErrorDistribution)
}
