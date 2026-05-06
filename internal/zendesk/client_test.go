package zendesk

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tukaelu/zgsync/internal/testutil"
)

func TestClientImpl_BaseURL(t *testing.T) {
	t.Parallel()

	client := NewClient("mycompany", "user@example.com", "token123")
	impl := client.(*clientImpl)

	expected := "https://mycompany.zendesk.com"
	actual := impl.baseURL()

	if actual != expected {
		t.Errorf("baseURL() = %s, want %s", actual, expected)
	}

	// ClientBaseURL is the exported wrapper used by CLI tests to verify subdomain propagation.
	if got := ClientBaseURL(client); got != expected {
		t.Errorf("ClientBaseURL() = %s, want %s", got, expected)
	}
}

func TestClientImpl_AuthorizationToken(t *testing.T) {
	t.Parallel()

	client := NewClient("test", "user@example.com/token", "secrettoken")
	impl := client.(*clientImpl)

	expected := "dXNlckBleGFtcGxlLmNvbS90b2tlbjpzZWNyZXR0b2tlbg=="
	actual := impl.authorizationToken()

	if actual != expected {
		t.Errorf("authorizationToken() = %s, want %s", actual, expected)
	}
}

func TestClientImpl_DoRequest_EmptyEndpoint(t *testing.T) {
	t.Parallel()

	client := &clientImpl{
		subdomain: "test",
		email:     "test@example.com/token",
		token:     "testtoken",
	}

	_, err := client.doRequest("GET", "", nil)

	errorChecker := testutil.NewErrorChecker(t)
	errorChecker.ExpectErrorContaining(err, "endpoint is required", "doRequest with empty endpoint")
}

// newTestClientImpl creates a production clientImpl configured to use a test server URL.
func newTestClientImpl(serverURL string) *clientImpl {
	return &clientImpl{
		subdomain:       "test",
		email:           "test@example.com/token",
		token:           "testtoken",
		baseURLOverride: serverURL,
	}
}

func TestClient_CreateArticle_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		locale         string
		sectionID      int
		payload        string
		serverStatus   int
		serverResponse string
		expectError    bool
		validateReq    func(*testing.T, *http.Request)
		validateResp   func(*testing.T, string)
	}{
		{
			name:         "successful article creation",
			locale:       "en_us",
			sectionID:    123,
			payload:      `{"article":{"title":"Test Article","locale":"en_us"}}`,
			serverStatus: http.StatusCreated,
			serverResponse: `{
				"article": {
					"id": 456,
					"title": "Test Article",
					"locale": "en_us",
					"section_id": 123
				}
			}`,
			expectError: false,
			validateReq: func(t *testing.T, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				expectedPath := "/api/v2/help_center/en_us/sections/123/articles.json"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Expected Basic auth header, got %s", auth)
				}

				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected application/json content type, got %s", contentType)
				}
			},
			validateResp: func(t *testing.T, resp string) {
				if !strings.Contains(resp, `"id": 456`) {
					t.Errorf("Response should contain article ID 456")
				}
				if !strings.Contains(resp, `"title": "Test Article"`) {
					t.Errorf("Response should contain article title")
				}
			},
		},
		{
			name:         "authentication failure",
			locale:       "ja",
			sectionID:    789,
			payload:      `{"article":{"title":"Test"}}`,
			serverStatus: http.StatusUnauthorized,
			serverResponse: `{
				"error": "Unauthorized",
				"description": "Authentication credentials invalid"
			}`,
			expectError:  true,
			validateReq:  func(t *testing.T, r *http.Request) {},
			validateResp: func(t *testing.T, resp string) {},
		},
		{
			name:         "malformed request",
			locale:       "fr",
			sectionID:    999,
			payload:      `{"article":{"invalid_field":"value"}}`,
			serverStatus: http.StatusBadRequest,
			serverResponse: `{
				"error": "BadRequest",
				"description": "Invalid article data"
			}`,
			expectError:  true,
			validateReq:  func(t *testing.T, r *http.Request) {},
			validateResp: func(t *testing.T, resp string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.validateReq(t, r)
				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := newTestClientImpl(server.URL)

			result, err := client.CreateArticle(tt.locale, tt.sectionID, tt.payload)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				tt.validateResp(t, result)
			}
		})
	}
}

func TestClient_ShowArticle_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		locale         string
		articleID      int
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:         "successful article retrieval",
			locale:       "en_us",
			articleID:    123,
			serverStatus: http.StatusOK,
			serverResponse: `{
				"article": {
					"id": 123,
					"title": "Existing Article",
					"locale": "en_us",
					"body": "<p>Article content</p>"
				}
			}`,
			expectError: false,
		},
		{
			name:         "article not found",
			locale:       "ja",
			articleID:    999,
			serverStatus: http.StatusNotFound,
			serverResponse: `{
				"error": "RecordNotFound",
				"description": "Article not found"
			}`,
			expectError: true,
		},
		{
			name:         "server error",
			locale:       "en_us",
			articleID:    456,
			serverStatus: http.StatusInternalServerError,
			serverResponse: `{
				"error": "InternalServerError",
				"description": "Server error occurred"
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET method, got %s", r.Method)
				}

				expectedPath := fmt.Sprintf("/api/v2/help_center/%s/articles/%d", tt.locale, tt.articleID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := newTestClientImpl(server.URL)

			result, err := client.ShowArticle(tt.locale, tt.articleID)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if !strings.Contains(result, fmt.Sprintf(`"id": %d`, tt.articleID)) {
					t.Errorf("Response should contain article ID %d", tt.articleID)
				}
			}
		})
	}
}

func TestClient_Translation_Operations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/translations") && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"translation": {
					"id": 789,
					"locale": "ja",
					"title": "Japanese Title",
					"source_id": 123
				}
			}`))
		case strings.Contains(r.URL.Path, "/translations/") && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"translation": {
					"id": 789,
					"locale": "ja",
					"title": "Updated Japanese Title",
					"source_id": 123
				}
			}`))
		case strings.Contains(r.URL.Path, "/translations/") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"translation": {
					"id": 789,
					"locale": "ja",
					"title": "Japanese Title",
					"source_id": 123,
					"body": "<h1>Japanese Content</h1>"
				}
			}`))
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClientImpl(server.URL)

	t.Run("CreateTranslation", func(t *testing.T) {
		payload := `{"translation":{"locale":"ja","title":"Japanese Title"}}`
		result, err := client.CreateTranslation(123, payload)
		if err != nil {
			t.Errorf("CreateTranslation failed: %v", err)
		}
		if !strings.Contains(result, `"id": 789`) {
			t.Errorf("Response should contain translation ID")
		}
	})

	t.Run("UpdateTranslation", func(t *testing.T) {
		payload := `{"translation":{"title":"Updated Japanese Title"}}`
		result, err := client.UpdateTranslation(123, "ja", payload)
		if err != nil {
			t.Errorf("UpdateTranslation failed: %v", err)
		}
		if !strings.Contains(result, `"title": "Updated Japanese Title"`) {
			t.Errorf("Response should contain updated title")
		}
	})

	t.Run("ShowTranslation", func(t *testing.T) {
		result, err := client.ShowTranslation(123, "ja")
		if err != nil {
			t.Errorf("ShowTranslation failed: %v", err)
		}
		if !strings.Contains(result, `"body": "<h1>Japanese Content</h1>"`) {
			t.Errorf("Response should contain translation body")
		}
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		operation   func(Client) error
		expectError string
	}{
		{
			name: "invalid JSON response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`invalid json response`))
				}))
			},
			operation: func(c Client) error {
				_, err := c.ShowArticle("en_us", 123)
				return err
			},
			expectError: "", // Should return response as-is, not parse JSON
		},
		{
			name: "server unavailable",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error": "Service temporarily unavailable"}`))
				}))
			},
			operation: func(c Client) error {
				_, err := c.ShowArticle("en_us", 123)
				return err
			},
			expectError: "unexpected status code: 503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := tt.setupServer()
			defer server.Close()

			client := newTestClientImpl(server.URL)

			err := tt.operation(client)

			if tt.expectError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got none", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			}
		})
	}
}

func TestClient_AdditionalErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		operation   func(*clientImpl) error
		expectError string
	}{
		{
			name: "request creation failure",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			operation: func(c *clientImpl) error {
				_, err := c.doRequest("INVALID\nMETHOD", "/valid/endpoint", nil)
				return err
			},
			expectError: "invalid method",
		},
		{
			name: "response body read failure simulation",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("{\"partial\":"))
					if hijacker, ok := w.(http.Hijacker); ok {
						conn, _, _ := hijacker.Hijack()
						_ = conn.Close()
					}
				}))
			},
			operation: func(c *clientImpl) error {
				_, err := c.ShowArticle("en_us", 123)
				return err
			},
			expectError: "EOF",
		},
		{
			name: "various HTTP status codes",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot) // 418 status
					_, _ = w.Write([]byte(`{"error": "I'm a teapot"}`))
				}))
			},
			operation: func(c *clientImpl) error {
				_, err := c.ShowArticle("en_us", 123)
				return err
			},
			expectError: "unexpected status code: 418",
		},
		{
			name: "network connection failure",
			setupServer: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				server.Close()
				return server
			},
			operation: func(c *clientImpl) error {
				_, err := c.ShowArticle("en_us", 123)
				return err
			},
			expectError: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := tt.setupServer()
			if tt.name != "network connection failure" {
				defer server.Close()
			}

			client := newTestClientImpl(server.URL)

			err := tt.operation(client)

			if tt.expectError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got none", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			}
		})
	}
}

func TestClient_APIResponseErrors(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantError    string
	}{
		{
			name:         "404 not found error",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "RecordNotFound"}`,
			wantError:    "unexpected status code: 404",
		},
		{
			name:         "422 unprocessable entity error",
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: `{"errors": {"title": ["can't be blank"]}}`,
			wantError:    "unexpected status code: 422",
		},
		{
			name:         "500 internal server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "InternalServerError"}`,
			wantError:    "unexpected status code: 500",
		},
		{
			name:         "503 service unavailable",
			statusCode:   http.StatusServiceUnavailable,
			responseBody: `{"error": "ServiceUnavailable"}`,
			wantError:    "unexpected status code: 503",
		},
		{
			name:         "502 bad gateway error",
			statusCode:   http.StatusBadGateway,
			responseBody: `{"error": "BadGateway"}`,
			wantError:    "unexpected status code: 502",
		},
		{
			name:         "504 gateway timeout error",
			statusCode:   http.StatusGatewayTimeout,
			responseBody: `{"error": "GatewayTimeout"}`,
			wantError:    "unexpected status code: 504",
		},
		{
			name:         "400 bad request",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": "BadRequest"}`,
			wantError:    "unexpected status code: 400",
		},
		{
			name:         "409 conflict error",
			statusCode:   http.StatusConflict,
			responseBody: `{"error": "Conflict"}`,
			wantError:    "unexpected status code: 409",
		},
		{
			name:         "405 method not allowed",
			statusCode:   http.StatusMethodNotAllowed,
			responseBody: `{"error": "MethodNotAllowed"}`,
			wantError:    "unexpected status code: 405",
		},
		{
			name:         "406 not acceptable",
			statusCode:   http.StatusNotAcceptable,
			responseBody: `{"error": "NotAcceptable"}`,
			wantError:    "unexpected status code: 406",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer ts.Close()

			client := newTestClientImpl(ts.URL)

			_, err := client.ShowArticle("en", 123)
			if err == nil {
				t.Errorf("ShowArticle() expected error for %s, got nil", tt.name)
			} else if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ShowArticle() error = %v, want error containing %v", err, tt.wantError)
			}
		})
	}
}

func TestClient_EmptyResponseBody(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantError    bool
	}{
		{
			name:         "empty body with 200 OK",
			statusCode:   http.StatusOK,
			responseBody: "",
			wantError:    false,
		},
		{
			name:         "empty body with 201 Created",
			statusCode:   http.StatusCreated,
			responseBody: "",
			wantError:    false,
		},
		{
			name:         "empty body with 204 No Content",
			statusCode:   http.StatusNoContent,
			responseBody: "",
			wantError:    true,
		},
		{
			name:         "whitespace only body",
			statusCode:   http.StatusOK,
			responseBody: "   \n\t   ",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer ts.Close()

			client := newTestClientImpl(ts.URL)

			result, err := client.ShowArticle("en", 123)
			if tt.wantError && err == nil {
				t.Errorf("ShowArticle() expected error for %s, got nil", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ShowArticle() unexpected error = %v", err)
			}
			if !tt.wantError && result != tt.responseBody {
				t.Errorf("ShowArticle() result = %q, want %q", result, tt.responseBody)
			}
		})
	}
}

func TestClient_NetworkErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		wantError   bool
	}{
		{
			name: "connection refused",
			setupServer: func() *httptest.Server {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				ts.Close()
				return ts
			},
			wantError: true,
		},
		{
			name: "server closes connection abruptly",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hj.Hijack()
						_ = conn.Close()
					}
				}))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := tt.setupServer()
			if ts != nil && tt.name != "connection refused" {
				defer ts.Close()
			}

			client := newTestClientImpl(ts.URL)

			_, err := client.ShowArticle("en", 123)
			if tt.wantError && err == nil {
				t.Errorf("ShowArticle() expected error for %s, got nil", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ShowArticle() unexpected error = %v", err)
			}
		})
	}
}

func TestClient_LargeResponseHandling(t *testing.T) {
	tests := []struct {
		name         string
		responseSize int
	}{
		{
			name:         "1MB response",
			responseSize: 1024 * 1024,
		},
		{
			name:         "10MB response",
			responseSize: 10 * 1024 * 1024,
		},
		{
			name:         "empty array response",
			responseSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)

				if tt.responseSize == 0 {
					_, _ = w.Write([]byte(`{"articles": []}`))
				} else {
					response := `{"data": "`
					response += strings.Repeat("x", tt.responseSize)
					response += `"}`
					_, _ = w.Write([]byte(response))
				}
			}))
			defer ts.Close()

			client := newTestClientImpl(ts.URL)

			result, err := client.ShowArticle("en", 123)
			if err != nil {
				t.Errorf("ShowArticle() error = %v", err)
			}

			if tt.responseSize == 0 {
				if result != `{"articles": []}` {
					t.Errorf("ShowArticle() unexpected result for empty response")
				}
			} else {
				expectedLen := len(`{"data": "`) + tt.responseSize + len(`"}`)
				if len(result) != expectedLen {
					t.Errorf("ShowArticle() result length = %d, want %d", len(result), expectedLen)
				}
			}
		})
	}
}

func TestClient_SpecialCharactersInPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "unicode characters",
			payload: `{"title": "記事タイトル 📝", "body": "これはテスト記事です。"}`,
		},
		{
			name:    "escaped characters",
			payload: `{"title": "Test \"Article\"", "body": "Line 1\nLine 2\tTabbed"}`,
		},
		{
			name:    "HTML entities",
			payload: `{"title": "Test &amp; Article", "body": "<p>Test &lt;content&gt;</p>"}`,
		},
		{
			name:    "emoji and special symbols",
			payload: `{"title": "Test 🚀 Article", "body": "Special chars: € £ ¥ • ™"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var receivedBody []byte
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				receivedBody = body

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"article": {"id": 123}}`))
			}))
			defer ts.Close()

			client := newTestClientImpl(ts.URL)

			_, err := client.UpdateArticle("en", 123, tt.payload)
			if err != nil {
				t.Errorf("UpdateArticle() error = %v", err)
			}

			if string(receivedBody) != tt.payload {
				t.Errorf("Received payload = %s, want %s", string(receivedBody), tt.payload)
			}
		})
	}
}

func TestClientImpl_RealHTTPClient(t *testing.T) {
	t.Parallel()

	client := NewClient("test", "test@example.com/token", "testtoken")
	realClient := client.(*clientImpl)

	t.Run("CreateArticle_RealImplementation", func(t *testing.T) {
		t.Parallel()

		baseURL := realClient.baseURL()
		expectedBaseURL := "https://test.zendesk.com"
		if baseURL != expectedBaseURL {
			t.Errorf("Expected baseURL %s, got %s", expectedBaseURL, baseURL)
		}

		authToken := realClient.authorizationToken()
		expectedToken := base64.StdEncoding.EncodeToString([]byte("test@example.com/token:testtoken"))
		if authToken != expectedToken {
			t.Errorf("Expected authToken %s, got %s", expectedToken, authToken)
		}
	})

}

func TestClient_AuthenticationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		token          string
		serverStatus   int
		serverResponse string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "401 unauthorized with detailed message",
			email:          "test@example.com/token",
			token:          "invalidtoken",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"error": "Unauthorized", "description": "Authentication credentials invalid"}`,
			expectError:    true,
			errorContains:  "401",
		},
		{
			name:           "401 with missing token",
			email:          "test@example.com/token",
			token:          "",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"error": "Unauthorized", "description": "API token is required"}`,
			expectError:    true,
			errorContains:  "401",
		},
		{
			name:           "401 with malformed credentials",
			email:          "invalid-email-format",
			token:          "token123",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"error": "Unauthorized", "description": "Malformed authorization header"}`,
			expectError:    true,
			errorContains:  "401",
		},
		{
			name:           "401 with expired token",
			email:          "test@example.com/token",
			token:          "expiredtoken",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"error": "Unauthorized", "description": "API token has expired"}`,
			expectError:    true,
			errorContains:  "401",
		},
		{
			name:           "401 with suspended account",
			email:          "suspended@example.com/token",
			token:          "validtoken",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"error": "Unauthorized", "description": "Account has been suspended"}`,
			expectError:    true,
			errorContains:  "401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					t.Error("Authorization header is missing")
				}
				if !strings.HasPrefix(authHeader, "Basic ") {
					t.Errorf("Authorization header should start with 'Basic ', got: %s", authHeader)
				}

				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := &clientImpl{
				subdomain:       "test",
				email:           tt.email,
				token:           tt.token,
				baseURLOverride: server.URL,
			}

			_, err := client.ShowArticle("en_us", 123)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.name)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s but got: %v", tt.name, err)
				}
			}
		})
	}
}

func TestClient_AuthorizationHeaderGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		token          string
		expectedHeader string
	}{
		{
			name:           "standard email with token suffix",
			email:          "user@example.com/token",
			token:          "abc123xyz",
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user@example.com/token:abc123xyz")),
		},
		{
			name:           "email without token suffix",
			email:          "user@example.com",
			token:          "abc123xyz",
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user@example.com:abc123xyz")),
		},
		{
			name:           "email with special characters",
			email:          "user+test@example.com/token",
			token:          "abc123xyz",
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user+test@example.com/token:abc123xyz")),
		},
		{
			name:           "token with special characters",
			email:          "user@example.com/token",
			token:          "abc/123+xyz=",
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user@example.com/token:abc/123+xyz=")),
		},
		{
			name:           "empty token",
			email:          "user@example.com/token",
			token:          "",
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user@example.com/token:")),
		},
		{
			name:           "very long token",
			email:          "user@example.com/token",
			token:          strings.Repeat("a", 100),
			expectedHeader: base64.StdEncoding.EncodeToString([]byte("user@example.com/token:" + strings.Repeat("a", 100))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("test", tt.email, tt.token)
			impl := client.(*clientImpl)

			actualHeader := impl.authorizationToken()

			if actualHeader != tt.expectedHeader {
				t.Errorf("Authorization header mismatch for %s", tt.name)
				t.Errorf("Expected: %s", tt.expectedHeader)
				t.Errorf("Actual:   %s", actualHeader)
			}
		})
	}
}

func TestClient_AllMethods_AuthenticationFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer server.Close()

	client := &clientImpl{
		subdomain:       "test",
		email:           "invalid@example.com/token",
		token:           "invalidtoken",
		baseURLOverride: server.URL,
	}

	t.Run("CreateArticle authentication failure", func(t *testing.T) {
		_, err := client.CreateArticle("en_us", 123, `{"article":{"title":"Test"}}`)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("UpdateArticle authentication failure", func(t *testing.T) {
		_, err := client.UpdateArticle("en_us", 456, `{"article":{"title":"Updated"}}`)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("ShowArticle authentication failure", func(t *testing.T) {
		_, err := client.ShowArticle("en_us", 789)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("CreateTranslation authentication failure", func(t *testing.T) {
		_, err := client.CreateTranslation(123, `{"translation":{"locale":"ja","title":"Test"}}`)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("UpdateTranslation authentication failure", func(t *testing.T) {
		_, err := client.UpdateTranslation(456, "ja", `{"translation":{"title":"Updated"}}`)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("ShowTranslation authentication failure", func(t *testing.T) {
		_, err := client.ShowTranslation(789, "ja")
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})

	t.Run("ArchiveArticle authentication failure", func(t *testing.T) {
		err := client.ArchiveArticle(123)
		if err == nil {
			t.Error("Expected authentication error but got none")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("Expected 401 error, got: %v", err)
		}
	})
}

func TestClient_BasicAuthHeaderFormat(t *testing.T) {
	t.Parallel()

	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"article":{"id":123}}`))
	}))
	defer server.Close()

	client := newTestClientImpl(server.URL)

	_, _ = client.ShowArticle("en_us", 123)

	if !strings.HasPrefix(capturedAuthHeader, "Basic ") {
		t.Errorf("Authorization header should start with 'Basic ', got: %s", capturedAuthHeader)
	}

	base64Part := strings.TrimPrefix(capturedAuthHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(base64Part)
	if err != nil {
		t.Errorf("Failed to decode base64 auth: %v", err)
	}

	expectedDecoded := "test@example.com/token:testtoken"
	if string(decoded) != expectedDecoded {
		t.Errorf("Decoded auth mismatch. Expected: %s, Got: %s", expectedDecoded, string(decoded))
	}
}

func TestClientImpl_ArchiveArticle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		articleID    int
		serverStatus int
		expectError  bool
	}{
		{
			name:         "successful archive returns no error",
			articleID:    123,
			serverStatus: http.StatusNoContent,
			expectError:  false,
		},
		{
			name:         "404 returns error",
			articleID:    999,
			serverStatus: http.StatusNotFound,
			expectError:  true,
		},
		{
			name:         "500 returns error",
			articleID:    456,
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE method, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v2/help_center/articles/%d", tt.articleID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := &clientImpl{
				subdomain:       "test",
				email:           "test@example.com/token",
				token:           "testtoken",
				baseURLOverride: server.URL,
			}

			err := client.ArchiveArticle(tt.articleID)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClientImpl_DoDeleteRequest_EmptyEndpoint(t *testing.T) {
	t.Parallel()

	client := &clientImpl{
		subdomain: "test",
		email:     "test@example.com/token",
		token:     "testtoken",
	}

	err := client.doDeleteRequest("")

	if err == nil {
		t.Error("Expected error for empty endpoint but got none")
	}
	if err != nil && !strings.Contains(err.Error(), "endpoint is required") {
		t.Errorf("Expected 'endpoint is required' error, got: %v", err)
	}
}

func TestClientImpl_DoDeleteRequest_NonNoContentResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		serverStatus int
		wantError    string
	}{
		{
			name:         "404 not found returns error",
			serverStatus: http.StatusNotFound,
			wantError:    "unexpected status code: 404",
		},
		{
			name:         "403 forbidden returns error",
			serverStatus: http.StatusForbidden,
			wantError:    "unexpected status code: 403",
		},
		{
			name:         "500 internal server error returns error",
			serverStatus: http.StatusInternalServerError,
			wantError:    "unexpected status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := newTestClientImpl(server.URL)

			err := client.doDeleteRequest("/api/v2/help_center/articles/123")

			if err == nil {
				t.Errorf("Expected error containing %q but got none", tt.wantError)
			} else if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing %q, got: %v", tt.wantError, err)
			}
		})
	}
}

func TestClient_UpdateArticle_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		locale         string
		articleID      int
		payload        string
		serverStatus   int
		serverResponse string
		expectError    bool
		validateResp   func(*testing.T, string)
	}{
		{
			name:      "success",
			locale:    "en_us",
			articleID: 456,
			payload:   `{"article":{"title":"Updated Article"}}`,
			serverStatus: http.StatusOK,
			serverResponse: `{
				"article": {
					"id": 456,
					"title": "Updated Article",
					"locale": "en_us"
				}
			}`,
			expectError: false,
			validateResp: func(t *testing.T, resp string) {
				if !strings.Contains(resp, `"id": 456`) {
					t.Errorf("Response should contain article ID 456")
				}
				if !strings.Contains(resp, `"title": "Updated Article"`) {
					t.Errorf("Response should contain updated title")
				}
			},
		},
		{
			name:           "not found",
			locale:         "en_us",
			articleID:      999,
			payload:        `{"article":{"title":"Updated Article"}}`,
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"error": "RecordNotFound", "description": "Article not found"}`,
			expectError:    true,
			validateResp:   func(t *testing.T, resp string) {},
		},
		{
			name:           "server error",
			locale:         "en_us",
			articleID:      456,
			payload:        `{"article":{"title":"Updated Article"}}`,
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error": "InternalServerError", "description": "Server error occurred"}`,
			expectError:    true,
			validateResp:   func(t *testing.T, resp string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT method, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v2/help_center/%s/articles/%d", tt.locale, tt.articleID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := newTestClientImpl(server.URL)

			result, err := client.UpdateArticle(tt.locale, tt.articleID, tt.payload)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError {
				tt.validateResp(t, result)
			}
		})
	}
}

func TestClient_UpdateTranslation_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		articleID      int
		locale         string
		payload        string
		serverStatus   int
		serverResponse string
		expectError    bool
		validateResp   func(*testing.T, string)
	}{
		{
			name:      "success",
			articleID: 123,
			locale:    "ja",
			payload:   `{"translation":{"title":"更新されたタイトル"}}`,
			serverStatus: http.StatusOK,
			serverResponse: `{
				"translation": {
					"id": 789,
					"locale": "ja",
					"title": "更新されたタイトル",
					"source_id": 123
				}
			}`,
			expectError: false,
			validateResp: func(t *testing.T, resp string) {
				if !strings.Contains(resp, `"id": 789`) {
					t.Errorf("Response should contain translation ID 789")
				}
				if !strings.Contains(resp, `"locale": "ja"`) {
					t.Errorf("Response should contain locale ja")
				}
			},
		},
		{
			name:           "not found",
			articleID:      999,
			locale:         "ja",
			payload:        `{"translation":{"title":"Updated"}}`,
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"error": "RecordNotFound", "description": "Translation not found"}`,
			expectError:    true,
			validateResp:   func(t *testing.T, resp string) {},
		},
		{
			name:           "server error",
			articleID:      123,
			locale:         "ja",
			payload:        `{"translation":{"title":"Updated"}}`,
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error": "InternalServerError", "description": "Server error occurred"}`,
			expectError:    true,
			validateResp:   func(t *testing.T, resp string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT method, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v2/help_center/articles/%d/translations/%s", tt.articleID, tt.locale)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := newTestClientImpl(server.URL)

			result, err := client.UpdateTranslation(tt.articleID, tt.locale, tt.payload)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError {
				tt.validateResp(t, result)
			}
		})
	}
}
