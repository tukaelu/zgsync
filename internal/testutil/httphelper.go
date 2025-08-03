package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// HTTPHelper provides utilities for HTTP testing
type HTTPHelper struct {
	t *testing.T
}

// NewHTTPHelper creates a new HTTPHelper instance
func NewHTTPHelper(t *testing.T) *HTTPHelper {
	return &HTTPHelper{t: t}
}

// HTTPTestCase represents a test case for HTTP operations
type HTTPTestCase struct {
	Name             string
	Method           string
	Path             string
	RequestBody      string
	ResponseStatus   int
	ResponseBody     string
	ValidateRequest  func(*testing.T, *http.Request)
	ValidateResponse func(*testing.T, string)
}

// CreateTestServer creates a test server with custom handler
func (hh *HTTPHelper) CreateTestServer(handler http.HandlerFunc) *httptest.Server {
	hh.t.Helper()
	server := httptest.NewServer(handler)
	hh.t.Cleanup(func() {
		server.Close()
	})
	return server
}

// CreateJSONResponse creates a JSON response string
func (hh *HTTPHelper) CreateJSONResponse(data interface{}) string {
	hh.t.Helper()
	jsonData, err := json.Marshal(data)
	if err != nil {
		hh.t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return string(jsonData)
}

// CreateMockServer creates a mock server with predefined responses
func (hh *HTTPHelper) CreateMockServer(responses map[string]HTTPResponse) *httptest.Server {
	hh.t.Helper()

	handler := func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		response, exists := responses[key]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Not Found"}`))
			return
		}

		// Set response headers
		for k, v := range response.Headers {
			w.Header().Set(k, v)
		}

		w.WriteHeader(response.StatusCode)
		_, _ = w.Write([]byte(response.Body))
	}

	return hh.CreateTestServer(handler)
}

// HTTPResponse represents a mock HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// NewHTTPResponse creates a new HTTPResponse
func NewHTTPResponse(statusCode int, body string) HTTPResponse {
	return HTTPResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
}

// RunHTTPTestCases runs a series of HTTP test cases
func (hh *HTTPHelper) RunHTTPTestCases(testCases []HTTPTestCase, createClient func(string) interface{}) {
	hh.t.Helper()

	for _, tc := range testCases {
		hh.t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request if specified
				if tc.ValidateRequest != nil {
					tc.ValidateRequest(t, r)
				}

				// Send response
				w.WriteHeader(tc.ResponseStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.ResponseBody))
			}))
			defer server.Close()

			// This would need to be implemented based on the specific client type
			// For now, it's a placeholder for the concept
			t.Logf("Would test %s %s against %s", tc.Method, tc.Path, server.URL)
		})
	}
}

// AssertHTTPStatus asserts that an HTTP response has the expected status
func (hh *HTTPHelper) AssertHTTPStatus(resp *http.Response, expectedStatus int) {
	hh.t.Helper()
	if resp.StatusCode != expectedStatus {
		hh.t.Errorf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
}

// AssertHTTPHeader asserts that an HTTP response has the expected header
func (hh *HTTPHelper) AssertHTTPHeader(resp *http.Response, headerName, expectedValue string) {
	hh.t.Helper()
	actualValue := resp.Header.Get(headerName)
	if actualValue != expectedValue {
		hh.t.Errorf("Expected header %s to be '%s', got '%s'", headerName, expectedValue, actualValue)
	}
}

// ReadHTTPBody reads the body from an HTTP response
func (hh *HTTPHelper) ReadHTTPBody(resp *http.Response) string {
	hh.t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		hh.t.Fatalf("Failed to read response body: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return string(body)
}

// AssertJSONResponse asserts that a response body is valid JSON and contains expected data
func (hh *HTTPHelper) AssertJSONResponse(body string, expectedData interface{}) {
	hh.t.Helper()

	var actualData interface{}
	if err := json.Unmarshal([]byte(body), &actualData); err != nil {
		hh.t.Fatalf("Response body is not valid JSON: %v", err)
	}

	expectedJSON, err := json.Marshal(expectedData)
	if err != nil {
		hh.t.Fatalf("Failed to marshal expected data: %v", err)
	}

	var expectedParsed interface{}
	if err := json.Unmarshal(expectedJSON, &expectedParsed); err != nil {
		hh.t.Fatalf("Failed to unmarshal expected data: %v", err)
	}

	ah := NewAssertionHelper(hh.t)
	ah.Equal(fmt.Sprintf("%+v", expectedParsed), fmt.Sprintf("%+v", actualData), "JSON response")
}

// CreateZendeskArticleResponse creates a mock Zendesk article response
func (hh *HTTPHelper) CreateZendeskArticleResponse(id int, title, locale string, sectionID int) string {
	response := map[string]interface{}{
		"article": map[string]interface{}{
			"id":         id,
			"title":      title,
			"locale":     locale,
			"section_id": sectionID,
		},
	}
	return hh.CreateJSONResponse(response)
}

// CreateZendeskTranslationResponse creates a mock Zendesk translation response
func (hh *HTTPHelper) CreateZendeskTranslationResponse(id int, locale, title string, sourceID int, body string) string {
	response := map[string]interface{}{
		"translation": map[string]interface{}{
			"id":        id,
			"locale":    locale,
			"title":     title,
			"source_id": sourceID,
			"body":      body,
		},
	}
	return hh.CreateJSONResponse(response)
}

// CreateZendeskErrorResponse creates a mock Zendesk error response
func (hh *HTTPHelper) CreateZendeskErrorResponse(errorType, description string) string {
	response := map[string]interface{}{
		"error":       errorType,
		"description": description,
	}
	return hh.CreateJSONResponse(response)
}

// ValidateBasicAuth validates that a request has proper basic authentication
func (hh *HTTPHelper) ValidateBasicAuth(expectedAuth string) func(*testing.T, *http.Request) {
	return func(t *testing.T, r *http.Request) {
		t.Helper()
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Expected Basic auth header, got %s", auth)
		}
		actualAuth := strings.TrimPrefix(auth, "Basic ")
		if actualAuth != expectedAuth {
			t.Errorf("Expected auth token %s, got %s", expectedAuth, actualAuth)
		}
	}
}

// ValidateContentType validates that a request has the expected content type
func (hh *HTTPHelper) ValidateContentType(expectedType string) func(*testing.T, *http.Request) {
	return func(t *testing.T, r *http.Request) {
		t.Helper()
		contentType := r.Header.Get("Content-Type")
		if contentType != expectedType {
			t.Errorf("Expected content type %s, got %s", expectedType, contentType)
		}
	}
}

// ValidateRequestMethod validates that a request uses the expected HTTP method
func (hh *HTTPHelper) ValidateRequestMethod(expectedMethod string) func(*testing.T, *http.Request) {
	return func(t *testing.T, r *http.Request) {
		t.Helper()
		if r.Method != expectedMethod {
			t.Errorf("Expected method %s, got %s", expectedMethod, r.Method)
		}
	}
}

// ValidateRequestPath validates that a request uses the expected path
func (hh *HTTPHelper) ValidateRequestPath(expectedPath string) func(*testing.T, *http.Request) {
	return func(t *testing.T, r *http.Request) {
		t.Helper()
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
	}
}
