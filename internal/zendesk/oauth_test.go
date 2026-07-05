package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newOAuthTestServer(t *testing.T, wantParams map[string]string, response string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/oauth/tokens" {
			t.Errorf("Expected path /oauth/tokens, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

		var params map[string]string
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		for k, want := range wantParams {
			if got := params[k]; got != want {
				t.Errorf("Expected param %s=%s, got %s", k, want, got)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func TestOAuthClient_ExchangeAuthorizationCode(t *testing.T) {
	t.Parallel()

	response := `{
		"access_token": "access123",
		"refresh_token": "refresh456",
		"token_type": "bearer",
		"scope": "hc:read hc:write",
		"expires_in": 1800,
		"refresh_token_expires_in": 2592000
	}`
	server := newOAuthTestServer(t, map[string]string{
		"grant_type":    "authorization_code",
		"code":          "authcode",
		"client_id":     "myclient",
		"redirect_uri":  "http://127.0.0.1:8080/callback",
		"code_verifier": "verifier123",
	}, response, http.StatusOK)
	defer server.Close()

	client := &OAuthClient{subdomain: "test", baseURLOverride: server.URL}
	token, err := client.ExchangeAuthorizationCode("myclient", "authcode", "http://127.0.0.1:8080/callback", "verifier123")
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode() error = %v", err)
	}

	if token.AccessToken != "access123" {
		t.Errorf("AccessToken = %s, want access123", token.AccessToken)
	}
	if token.RefreshToken != "refresh456" {
		t.Errorf("RefreshToken = %s, want refresh456", token.RefreshToken)
	}
	if token.ExpiresIn != 1800 {
		t.Errorf("ExpiresIn = %d, want 1800", token.ExpiresIn)
	}
	if token.RefreshTokenExpiresIn != 2592000 {
		t.Errorf("RefreshTokenExpiresIn = %d, want 2592000", token.RefreshTokenExpiresIn)
	}
}

func TestOAuthClient_RefreshToken(t *testing.T) {
	t.Parallel()

	response := `{
		"access_token": "newaccess",
		"refresh_token": "newrefresh",
		"token_type": "bearer",
		"expires_in": 1800
	}`
	server := newOAuthTestServer(t, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": "oldrefresh",
		"client_id":     "myclient",
	}, response, http.StatusOK)
	defer server.Close()

	client := &OAuthClient{subdomain: "test", baseURLOverride: server.URL}
	token, err := client.RefreshToken("myclient", "oldrefresh")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if token.AccessToken != "newaccess" {
		t.Errorf("AccessToken = %s, want newaccess", token.AccessToken)
	}
	if token.RefreshToken != "newrefresh" {
		t.Errorf("RefreshToken = %s, want newrefresh", token.RefreshToken)
	}
}

func TestOAuthClient_ClientCredentials(t *testing.T) {
	t.Parallel()

	response := `{
		"access_token": "ciaccess",
		"token_type": "bearer",
		"scope": "hc:read hc:write",
		"expires_in": 1800
	}`
	server := newOAuthTestServer(t, map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     "ciclient",
		"client_secret": "cisecret",
		"scope":         "hc:read hc:write",
	}, response, http.StatusOK)
	defer server.Close()

	client := &OAuthClient{subdomain: "test", baseURLOverride: server.URL}
	token, err := client.ClientCredentials("ciclient", "cisecret", "hc:read hc:write")
	if err != nil {
		t.Fatalf("ClientCredentials() error = %v", err)
	}

	if token.AccessToken != "ciaccess" {
		t.Errorf("AccessToken = %s, want ciaccess", token.AccessToken)
	}
	if token.RefreshToken != "" {
		t.Errorf("RefreshToken = %s, want empty", token.RefreshToken)
	}
}

func TestOAuthClient_TokenRequestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        int
		response      string
		errorContains string
	}{
		{
			name:          "invalid grant returns 400",
			status:        http.StatusBadRequest,
			response:      `{"error": "invalid_grant", "error_description": "authorization code is invalid or expired"}`,
			errorContains: "status 400",
		},
		{
			name:          "unauthorized client returns 401",
			status:        http.StatusUnauthorized,
			response:      `{"error": "invalid_client"}`,
			errorContains: "status 401",
		},
		{
			name:          "missing access token in response",
			status:        http.StatusOK,
			response:      `{"token_type": "bearer"}`,
			errorContains: "does not contain an access token",
		},
		{
			// Defensive path for a Zendesk-side bug; unlikely in practice
			// but the branch must not panic.
			name:          "malformed JSON in 200 response",
			status:        http.StatusOK,
			response:      `{not valid json`,
			errorContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newOAuthTestServer(t, nil, tt.response, tt.status)
			defer server.Close()

			client := &OAuthClient{subdomain: "test", baseURLOverride: server.URL}
			_, err := client.RefreshToken("myclient", "refresh")

			if err == nil {
				t.Fatal("Expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

func TestOAuthTokenResponse_ExpiryTime(t *testing.T) {
	t.Parallel()

	// within asserts got is in [want-tolerance, want+tolerance].
	within := func(t *testing.T, got, want time.Time, tolerance time.Duration) {
		t.Helper()
		if got.Before(want.Add(-tolerance)) || got.After(want.Add(tolerance)) {
			t.Errorf("ExpiryTime() = %v, want %v +/- %v", got, want, tolerance)
		}
	}

	t.Run("zero expires_in means no expiry", func(t *testing.T) {
		t.Parallel()

		res := &OAuthTokenResponse{ExpiresIn: 0}
		if got := res.ExpiryTime(); !got.IsZero() {
			t.Errorf("ExpiryTime() = %v, want zero time", got)
		}
	})

	t.Run("negative expires_in means no expiry", func(t *testing.T) {
		t.Parallel()

		res := &OAuthTokenResponse{ExpiresIn: -1}
		if got := res.ExpiryTime(); !got.IsZero() {
			t.Errorf("ExpiryTime() = %v, want zero time", got)
		}
	})

	t.Run("margin is subtracted from the lifetime", func(t *testing.T) {
		t.Parallel()

		res := &OAuthTokenResponse{ExpiresIn: 1800}
		within(t, res.ExpiryTime(), time.Now().Add(1800*time.Second-tokenExpiryMargin), 2*time.Second)
	})

	t.Run("margin is clamped to half the lifetime for short-lived tokens", func(t *testing.T) {
		t.Parallel()

		res := &OAuthTokenResponse{ExpiresIn: 10}
		within(t, res.ExpiryTime(), time.Now().Add(5*time.Second), 2*time.Second)
	})
}

// TestNewOAuthClientWithBaseURL exists to keep the function covered in the
// zendesk package's own test binary. The function is only called from cli
// package tests, which do not instrument zendesk package code.
func TestNewOAuthClientWithBaseURL(t *testing.T) {
	t.Parallel()

	client := NewOAuthClientWithBaseURL("mycompany", "http://example.com")
	if client.subdomain != "mycompany" || client.baseURLOverride != "http://example.com" {
		t.Errorf("NewOAuthClientWithBaseURL() = %+v", client)
	}
	if got := client.baseURL(); got != "http://example.com" {
		t.Errorf("baseURL() = %s, want http://example.com", got)
	}
}

func TestOAuthClient_BaseURL(t *testing.T) {
	t.Parallel()

	client := NewOAuthClient("mycompany")
	expected := "https://mycompany.zendesk.com"
	if got := client.baseURL(); got != expected {
		t.Errorf("baseURL() = %s, want %s", got, expected)
	}
}
