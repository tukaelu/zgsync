package zendesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AdvancedMockServer provides a sophisticated mock implementation of Zendesk Help Center API
type AdvancedMockServer struct {
	server        *httptest.Server
	dataStore     *MockDataStore
	scenarios     *ScenarioManager
	errorSim      *ErrorSimulator
	errorTracker  *ErrorTracker
	config        *MockServerConfig
	requestLog    []RequestLog
	mutex         sync.RWMutex
}

// MockServerConfig controls server behavior
type MockServerConfig struct {
	BaseLatency       time.Duration
	ErrorRate         float64
	RateLimit         int
	EnableLogging     bool
	StrictMode        bool // Validate request format strictly
	EnableErrorSim    bool // Enable realistic error simulation
	ErrorScenarios    []string // List of error scenarios to enable
}

// RequestLog tracks all requests for debugging and verification
type RequestLog struct {
	Method    string
	Path      string
	Headers   map[string]string
	Body      string
	Timestamp time.Time
	Response  ResponseLog
}

type ResponseLog struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	Duration   time.Duration
}

// MockDataStore manages stateful data for realistic API simulation
type MockDataStore struct {
	articles     map[int]*Article
	translations map[string]*Translation // key: "{articleID}-{locale}"
	sections     map[int]*MockSection
	users        map[int]*MockUser
	nextID       struct {
		article     int
		translation int
		section     int
		user        int
	}
	mutex sync.RWMutex
}

// MockSection represents a section in the mock data store
type MockSection struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  int    `json:"category_id"`
	Locale      string `json:"locale"`
	Position    int    `json:"position"`
}

// MockUser represents a user in the mock data store
type MockUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ScenarioManager controls different behavior scenarios
type ScenarioManager struct {
	activeScenario string
	behaviors      map[string]*BehaviorSet
	mutex          sync.RWMutex
}

// BehaviorSet defines how the server behaves under a specific scenario
type BehaviorSet struct {
	Name            string
	ErrorRate       float64
	LatencyMin      time.Duration
	LatencyMax      time.Duration
	FailurePatterns map[string]ErrorPattern
	CustomHandlers  map[string]http.HandlerFunc
}

// ErrorPattern defines when and how to return errors
type ErrorPattern struct {
	StatusCode int
	Response   string
	Condition  func(*http.Request) bool
}

// NewAdvancedMockServer creates a new advanced mock server
func NewAdvancedMockServer(config *MockServerConfig) *AdvancedMockServer {
	if config == nil {
		config = &MockServerConfig{
			BaseLatency:   10 * time.Millisecond,
			ErrorRate:     0.0,
			RateLimit:     1000,
			EnableLogging: true,
			StrictMode:    false,
		}
	}

	dataStore := &MockDataStore{
		articles:     make(map[int]*Article),
		translations: make(map[string]*Translation),
		sections:     make(map[int]*MockSection),
		users:        make(map[int]*MockUser),
	}

	// Initialize with default data
	dataStore.initializeDefaultData()

	scenarios := &ScenarioManager{
		activeScenario: "Normal",
		behaviors:      make(map[string]*BehaviorSet),
	}

	// Initialize default scenarios
	scenarios.initializeDefaultScenarios()

	// Initialize error simulation components
	errorSim := NewErrorSimulator()
	errorTracker := NewErrorTracker()

	server := &AdvancedMockServer{
		dataStore:    dataStore,
		scenarios:    scenarios,
		errorSim:     errorSim,
		errorTracker: errorTracker,
		config:       config,
		requestLog:   make([]RequestLog, 0),
	}

	// Create HTTP server with custom mux
	mux := http.NewServeMux()
	server.registerRoutes(mux)

	server.server = httptest.NewServer(http.HandlerFunc(server.middleware(mux)))

	return server
}

// URL returns the base URL of the mock server
func (s *AdvancedMockServer) URL() string {
	return s.server.URL
}

// Close shuts down the mock server
func (s *AdvancedMockServer) Close() {
	s.server.Close()
}

// SetScenario changes the active behavior scenario
func (s *AdvancedMockServer) SetScenario(name string) error {
	s.scenarios.mutex.Lock()
	defer s.scenarios.mutex.Unlock()

	if _, exists := s.scenarios.behaviors[name]; !exists {
		return fmt.Errorf("scenario %s not found", name)
	}

	s.scenarios.activeScenario = name
	return nil
}

// GetRequestLog returns all logged requests
func (s *AdvancedMockServer) GetRequestLog() []RequestLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return copy to prevent external modification
	log := make([]RequestLog, len(s.requestLog))
	copy(log, s.requestLog)
	return log
}

// ClearRequestLog clears all logged requests
func (s *AdvancedMockServer) ClearRequestLog() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.requestLog = s.requestLog[:0]
}

// middleware applies logging, scenario behaviors, and other cross-cutting concerns
func (s *AdvancedMockServer) middleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Log request if enabled
		var requestLog *RequestLog
		if s.config.EnableLogging {
			requestLog = &RequestLog{
				Method:    r.Method,
				Path:      r.URL.Path,
				Headers:   make(map[string]string),
				Timestamp: startTime,
			}

			// Copy headers
			for key, values := range r.Header {
				if len(values) > 0 {
					requestLog.Headers[key] = values[0]
				}
			}

			// Read body for logging (note: this consumes the body)
			if r.Body != nil {
				// For simplicity, we'll skip body logging to avoid consumption issues
				// In a production mock server, you'd want to use io.TeeReader
			}
		}

		// Create response writer wrapper for logging
		rw := &responseWriter{ResponseWriter: w}

		// Apply scenario behaviors (may write response and return early)
		if s.applyScenarioBehavior(rw, r) {
			// Scenario behavior handled the response, skip normal handler
		} else {
			// Call the actual handler
			next.ServeHTTP(rw, r)
		}

		// Complete request logging
		if requestLog != nil {
			requestLog.Response = ResponseLog{
				StatusCode: rw.statusCode,
				Duration:   time.Since(startTime),
				Headers:    make(map[string]string),
			}

			// Copy response headers
			for key, values := range w.Header() {
				if len(values) > 0 {
					requestLog.Response.Headers[key] = values[0]
				}
			}

			s.mutex.Lock()
			s.requestLog = append(s.requestLog, *requestLog)
			s.mutex.Unlock()
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// applyScenarioBehavior applies the current scenario's behavior
// Returns true if the scenario handled the response, false if normal processing should continue
func (s *AdvancedMockServer) applyScenarioBehavior(w http.ResponseWriter, r *http.Request) bool {
	s.scenarios.mutex.RLock()
	behavior := s.scenarios.behaviors[s.scenarios.activeScenario]
	activeScenario := s.scenarios.activeScenario
	s.scenarios.mutex.RUnlock()

	if behavior == nil {
		return false
	}

	// Apply realistic error simulation if enabled
	if s.config.EnableErrorSim {
		if s.checkRealisticErrors(w, r, activeScenario) {
			return true // Error was simulated, stop processing
		}
	}

	// Apply latency
	if behavior.LatencyMin > 0 {
		latency := behavior.LatencyMin
		if behavior.LatencyMax > behavior.LatencyMin {
			// Add random component for realistic simulation
			extra := time.Duration(float64(behavior.LatencyMax-behavior.LatencyMin) * 0.5) // Simplified
			latency += extra
		}
		time.Sleep(latency)
	}

	// Check for failure patterns
	for pattern, errorPattern := range behavior.FailurePatterns {
		if errorPattern.Condition != nil && errorPattern.Condition(r) {
			http.Error(w, errorPattern.Response, errorPattern.StatusCode)
			return true
		}
		_ = pattern // avoid unused variable error
	}

	return false
}

// checkRealisticErrors applies realistic error simulation using ErrorSimulator
func (s *AdvancedMockServer) checkRealisticErrors(w http.ResponseWriter, r *http.Request, activeScenario string) bool {
	// Check configured error scenarios
	errorScenarios := s.config.ErrorScenarios
	if len(errorScenarios) == 0 {
		// Default to checking all available scenarios
		errorScenarios = s.errorSim.GetAvailableScenarios()
	}

	for _, scenarioName := range errorScenarios {
		s.errorTracker.RecordCheck(scenarioName, false) // Default to no error

		if errorSim := s.errorSim.SimulateError(r, scenarioName); errorSim != nil {
			// Record the error occurrence
			s.errorTracker.RecordCheck(scenarioName, true)

			// Apply backoff delay if specified
			if errorSim.BackoffDelay > 0 {
				time.Sleep(errorSim.BackoffDelay)
			}

			// Set appropriate headers for specific error types
			s.setErrorHeaders(w, errorSim.StatusCode)

			// Set status code and write response body
			w.WriteHeader(errorSim.StatusCode)
			w.Write([]byte(errorSim.Response))
			return true
		}
	}

	return false
}

// setErrorHeaders sets appropriate headers for different error types
func (s *AdvancedMockServer) setErrorHeaders(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	
	switch statusCode {
	case http.StatusTooManyRequests:
		w.Header().Set("Retry-After", "30")
		w.Header().Set("X-Rate-Limit-Limit", "100")
		w.Header().Set("X-Rate-Limit-Remaining", "0")
	case http.StatusUnauthorized:
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Zendesk API\"")
	case http.StatusServiceUnavailable:
		w.Header().Set("Retry-After", "300")
	}
}

// Error simulation configuration methods

// EnableErrorSimulation enables realistic error simulation
func (s *AdvancedMockServer) EnableErrorSimulation(scenarios []string) {
	s.config.EnableErrorSim = true
	s.config.ErrorScenarios = scenarios
}

// DisableErrorSimulation disables error simulation
func (s *AdvancedMockServer) DisableErrorSimulation() {
	s.config.EnableErrorSim = false
	s.config.ErrorScenarios = nil
}

// GetErrorDistributions returns error occurrence statistics
func (s *AdvancedMockServer) GetErrorDistributions() map[string]*ErrorDistribution {
	return s.errorTracker.GetDistributions()
}

// ResetErrorTracking clears all error statistics
func (s *AdvancedMockServer) ResetErrorTracking() {
	s.errorTracker.Reset()
}

// AddCustomErrorScenario adds a custom error scenario to the simulator
func (s *AdvancedMockServer) AddCustomErrorScenario(name string, scenario *ErrorScenario) {
	s.errorSim.AddCustomScenario(name, scenario)
}

// GetAvailableErrorScenarios returns all available error scenarios
func (s *AdvancedMockServer) GetAvailableErrorScenarios() []string {
	return s.errorSim.GetAvailableScenarios()
}

// CreateCompositeErrorScenario creates a scenario combining multiple error types
func (s *AdvancedMockServer) CreateCompositeErrorScenario(name string, scenarios []string, probability float64) {
	s.errorSim.CreateCompositeScenario(name, scenarios, probability)
}

// registerRoutes sets up all API endpoints
func (s *AdvancedMockServer) registerRoutes(mux *http.ServeMux) {
	// Article endpoints
	mux.HandleFunc("/api/v2/help_center/", s.handleHelpCenterRequest)
}

// handleHelpCenterRequest routes help center API requests
func (s *AdvancedMockServer) handleHelpCenterRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/")
	
	// Route to appropriate handler based on path pattern
	switch {
	case strings.HasSuffix(path, "/articles.json") && r.Method == "POST":
		s.handleCreateArticle(w, r)
	case strings.Contains(path, "/articles/") && strings.HasSuffix(path, ".json") && r.Method == "GET":
		s.handleShowArticle(w, r)
	case strings.Contains(path, "/articles/") && r.Method == "PUT":
		s.handleUpdateArticle(w, r)
	case strings.Contains(path, "/translations.json") && r.Method == "POST":
		s.handleCreateTranslation(w, r)
	case strings.Contains(path, "/translations/") && r.Method == "PUT":
		s.handleUpdateTranslation(w, r)
	case strings.Contains(path, "/translations/") && r.Method == "GET":
		s.handleShowTranslation(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleCreateArticle handles POST /api/v2/help_center/{locale}/sections/{section_id}/articles.json
func (s *AdvancedMockServer) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract locale and section_id from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	locale := pathParts[0]
	sectionIDStr := pathParts[2]
	
	sectionID, err := strconv.Atoi(sectionIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid section ID"}`, http.StatusBadRequest)
		return
	}

	// Verify section exists
	if !s.dataStore.sectionExists(sectionID) {
		http.Error(w, `{"error": "Section not found"}`, http.StatusNotFound)
		return
	}

	// Create new article
	article := s.dataStore.createArticle(locale, sectionID)
	
	// Return created article
	response := map[string]*Article{"article": article}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleShowArticle handles GET /api/v2/help_center/{locale}/articles/{id}.json
func (s *AdvancedMockServer) handleShowArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract article ID from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	articleIDStr := strings.TrimSuffix(pathParts[2], ".json")
	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid article ID"}`, http.StatusBadRequest)
		return
	}

	// Get article
	article := s.dataStore.getArticle(articleID)
	if article == nil {
		http.Error(w, `{"error": "Article not found"}`, http.StatusNotFound)
		return
	}

	// Return article
	response := map[string]*Article{"article": article}
	json.NewEncoder(w).Encode(response)
}

// handleUpdateArticle handles PUT /api/v2/help_center/{locale}/articles/{id}
func (s *AdvancedMockServer) handleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Extract article ID from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	articleIDStr := pathParts[2]
	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid article ID"}`, http.StatusBadRequest)
		return
	}

	// Update article
	article := s.dataStore.updateArticle(articleID)
	if article == nil {
		http.Error(w, `{"error": "Article not found"}`, http.StatusNotFound)
		return
	}

	// Return updated article
	response := map[string]*Article{"article": article}
	json.NewEncoder(w).Encode(response)
}

// handleCreateTranslation handles POST /api/v2/help_center/articles/{article_id}/translations.json
func (s *AdvancedMockServer) handleCreateTranslation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract article ID from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/articles/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	articleIDStr := pathParts[0]
	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid article ID"}`, http.StatusBadRequest)
		return
	}

	// Verify article exists
	if !s.dataStore.articleExists(articleID) {
		http.Error(w, `{"error": "Article not found"}`, http.StatusNotFound)
		return
	}

	// Create translation
	translation := s.dataStore.createTranslation(articleID, "ja") // Default locale for now
	
	// Return created translation
	response := map[string]*Translation{"translation": translation}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleUpdateTranslation handles PUT /api/v2/help_center/articles/{article_id}/translations/{locale}
func (s *AdvancedMockServer) handleUpdateTranslation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract article ID and locale from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/articles/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	articleIDStr := pathParts[0]
	locale := pathParts[2]
	
	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid article ID"}`, http.StatusBadRequest)
		return
	}

	// Update translation
	translation := s.dataStore.updateTranslation(articleID, locale)
	if translation == nil {
		http.Error(w, `{"error": "Translation not found"}`, http.StatusNotFound)
		return
	}

	// Return updated translation
	response := map[string]*Translation{"translation": translation}
	json.NewEncoder(w).Encode(response)
}

// handleShowTranslation handles GET /api/v2/help_center/articles/{article_id}/translations/{locale}
func (s *AdvancedMockServer) handleShowTranslation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract article ID and locale from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/help_center/articles/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, `{"error": "Invalid URL format"}`, http.StatusBadRequest)
		return
	}

	articleIDStr := pathParts[0]
	locale := pathParts[2]
	
	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid article ID"}`, http.StatusBadRequest)
		return
	}

	// Get translation
	translation := s.dataStore.getTranslation(articleID, locale)
	if translation == nil {
		http.Error(w, `{"error": "Translation not found"}`, http.StatusNotFound)
		return
	}

	// Return translation
	response := map[string]*Translation{"translation": translation}
	json.NewEncoder(w).Encode(response)
}