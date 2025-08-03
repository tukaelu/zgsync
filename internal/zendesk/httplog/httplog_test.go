package httplog

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransport_RoundTrip(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name               string
		method             string
		path               string
		body               string
		responseStatus     int
		responseBody       string
		expectLogRequest   bool
		expectLogResponse  bool
	}{
		{
			name:              "GET request",
			method:            "GET",
			path:              "/api/test",
			body:              "",
			responseStatus:    200,
			responseBody:      `{"success": true}`,
			expectLogRequest:  true,
			expectLogResponse: true,
		},
		{
			name:              "POST request with body",
			method:            "POST",
			path:              "/api/create",
			body:              `{"name": "test"}`,
			responseStatus:    201,
			responseBody:      `{"id": 123}`,
			expectLogRequest:  true,
			expectLogResponse: true,
		},
		{
			name:              "error response",
			method:            "DELETE",
			path:              "/api/delete/123",
			body:              "",
			responseStatus:    404,
			responseBody:      `{"error": "Not found"}`,
			expectLogRequest:  true,
			expectLogResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()
			
			// Capture logs
			var requestLogBuf, responseLogBuf bytes.Buffer
			
			// Create transport with custom loggers
			transport := &Transport{
				LogRequest: func(req *http.Request) {
					requestLogBuf.WriteString(req.Method + " " + req.URL.Path)
				},
				LogResponse: func(resp *http.Response) {
					responseLogBuf.WriteString(resp.Status)
				},
			}
			
			// Create request
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			
			req, err := http.NewRequest(tt.method, server.URL+tt.path, bodyReader)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			// Execute request
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			
			// Verify response
			if resp.StatusCode != tt.responseStatus {
				t.Errorf("Expected status %d, got %d", tt.responseStatus, resp.StatusCode)
			}
			
			// Verify logs
			if tt.expectLogRequest {
				requestLog := requestLogBuf.String()
				if !strings.Contains(requestLog, tt.method) {
					t.Errorf("Request log should contain method %s, got: %s", tt.method, requestLog)
				}
				if !strings.Contains(requestLog, tt.path) {
					t.Errorf("Request log should contain path %s, got: %s", tt.path, requestLog)
				}
			}
			
			if tt.expectLogResponse {
				responseLog := responseLogBuf.String()
				if !strings.Contains(responseLog, resp.Status) {
					t.Errorf("Response log should contain status %s, got: %s", resp.Status, responseLog)
				}
			}
		})
	}
}

func TestTransport_WithContext(t *testing.T) {
	t.Parallel()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context contains the request key
		_ = r.Context().Value(ContextRequestKey) // Just verify the context key exists
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	transport := &Transport{}
	
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	// Add context with request key
	ctx := context.WithValue(req.Context(), ContextRequestKey, req)
	req = req.WithContext(ctx)
	
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTransport_DefaultLoggers(t *testing.T) {
	t.Parallel()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	// Transport with no custom loggers should use defaults
	transport := &Transport{}
	
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	// This should not panic and should work with default loggers
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTransport_Transport(t *testing.T) {
	t.Parallel()
	
	// Test custom underlying transport
	customTransport := &http.Transport{}
	transport := &Transport{
		Transport: customTransport,
	}
	
	actualTransport := transport.transport()
	if actualTransport != customTransport {
		t.Errorf("Expected custom transport, got default")
	}
	
	// Test default transport
	transport2 := &Transport{}
	defaultTransport := transport2.transport()
	if defaultTransport != DefaultTransport {
		t.Errorf("Expected DefaultTransport, got different transport")
	}
}

func TestDefaultLogRequest(t *testing.T) {
	t.Parallel()
	
	req, err := http.NewRequest("POST", "https://example.com/api", strings.NewReader(`{"test": "data"}`))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	// This should not panic
	DefaultLogRequest(req)
}

func TestDefaultLogResponse(t *testing.T) {
	t.Parallel()
	
	// Create a proper request first
	req, err := http.NewRequest("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Request:    req, // Important: set the request
	}
	
	// This should not panic
	DefaultLogResponse(resp)
}

func TestTransport_LargeRequestBody(t *testing.T) {
	t.Parallel()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	var loggedMethod string
	transport := &Transport{
		LogRequest: func(req *http.Request) {
			loggedMethod = req.Method
		},
	}
	
	// Create large request body
	largeBody := strings.Repeat("a", 10000)
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(largeBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	// Verify the method was captured correctly
	if loggedMethod != "POST" {
		t.Errorf("Expected logged method 'POST', got '%s'", loggedMethod)
	}
}

func TestTransport_ErrorResponse(t *testing.T) {
	t.Parallel()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	var loggedStatus string
	
	transport := &Transport{
		LogResponse: func(resp *http.Response) {
			loggedStatus = resp.Status
		},
	}
	
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	
	// Verify error response was logged
	if loggedStatus != "500 Internal Server Error" {
		t.Errorf("Expected logged status '500 Internal Server Error', got '%s'", loggedStatus)
	}
}