package zendesk

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthCredentials_AuthorizationHeader_ValidToken(t *testing.T) {
	t.Parallel()

	creds := &OAuthCredentials{
		AccessToken: "validtoken",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	authz, err := creds.AuthorizationHeader()
	if err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	if authz != "Bearer validtoken" {
		t.Errorf("AuthorizationHeader() = %s, want Bearer validtoken", authz)
	}
}

func TestOAuthCredentials_AuthorizationHeader_NoExpiry(t *testing.T) {
	t.Parallel()

	creds := &OAuthCredentials{AccessToken: "legacytoken"}

	authz, err := creds.AuthorizationHeader()
	if err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	if authz != "Bearer legacytoken" {
		t.Errorf("AuthorizationHeader() = %s, want Bearer legacytoken", authz)
	}
}

func TestOAuthCredentials_AuthorizationHeader_RefreshesExpiredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "refreshedtoken",
			"refresh_token": "newrefresh",
			"token_type": "bearer",
			"expires_in": 1800
		}`))
	}))
	defer server.Close()

	var savedAccess, savedRefresh string
	var savedExpiry time.Time
	creds := &OAuthCredentials{
		AccessToken:  "expiredtoken",
		RefreshToken: "oldrefresh",
		Expiry:       time.Now().Add(-1 * time.Minute),
		ClientID:     "myclient",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
		OnRefresh: func(accessToken, refreshToken string, expiry time.Time) error {
			savedAccess = accessToken
			savedRefresh = refreshToken
			savedExpiry = expiry
			return nil
		},
	}

	authz, err := creds.AuthorizationHeader()
	if err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	if authz != "Bearer refreshedtoken" {
		t.Errorf("AuthorizationHeader() = %s, want Bearer refreshedtoken", authz)
	}
	if savedAccess != "refreshedtoken" || savedRefresh != "newrefresh" {
		t.Errorf("OnRefresh received (%s, %s), want (refreshedtoken, newrefresh)", savedAccess, savedRefresh)
	}
	if savedExpiry.IsZero() {
		t.Error("OnRefresh received zero expiry")
	}
}

func TestOAuthCredentials_AuthorizationHeader_ExpiredWithoutRefreshToken(t *testing.T) {
	t.Parallel()

	creds := &OAuthCredentials{
		AccessToken: "expiredtoken",
		Expiry:      time.Now().Add(-1 * time.Minute),
	}

	_, err := creds.AuthorizationHeader()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected error to suggest `auth login`, got: %v", err)
	}
}

func TestOAuthCredentials_AuthorizationHeader_RefreshFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	creds := &OAuthCredentials{
		AccessToken:  "expiredtoken",
		RefreshToken: "expiredrefresh",
		Expiry:       time.Now().Add(-1 * time.Minute),
		ClientID:     "myclient",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	_, err := creds.AuthorizationHeader()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected error to suggest `auth login`, got: %v", err)
	}
}

func TestClientCredentialsProvider_FetchesAndCachesToken(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "citoken",
			"token_type": "bearer",
			"expires_in": 1800
		}`))
	}))
	defer server.Close()

	provider := &ClientCredentialsProvider{
		ClientID:     "ciclient",
		ClientSecret: "cisecret",
		Scope:        "hc:read hc:write",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	for i := 0; i < 3; i++ {
		authz, err := provider.AuthorizationHeader()
		if err != nil {
			t.Fatalf("AuthorizationHeader() error = %v", err)
		}
		if authz != "Bearer citoken" {
			t.Errorf("AuthorizationHeader() = %s, want Bearer citoken", authz)
		}
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("Expected 1 token request (cached afterwards), got %d", got)
	}
}

func TestClientCredentialsProvider_RefetchesExpiredToken(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "citoken",
			"token_type": "bearer",
			"expires_in": 1800
		}`))
	}))
	defer server.Close()

	provider := &ClientCredentialsProvider{
		ClientID:     "ciclient",
		ClientSecret: "cisecret",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	if _, err := provider.AuthorizationHeader(); err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	provider.expiry = time.Now().Add(-1 * time.Second)
	if _, err := provider.AuthorizationHeader(); err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("Expected 2 token requests after forced expiry, got %d", got)
	}
}

func TestClientCredentialsProvider_ShortLivedTokenIsNotThrashResistant(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		// Shorter than the expiry margin; the clamped margin must still leave
		// the token usable so consecutive requests do not refetch every time.
		_, _ = w.Write([]byte(`{
			"access_token": "citoken",
			"token_type": "bearer",
			"expires_in": 10
		}`))
	}))
	defer server.Close()

	provider := &ClientCredentialsProvider{
		ClientID:     "ciclient",
		ClientSecret: "cisecret",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	for i := 0; i < 3; i++ {
		if _, err := provider.AuthorizationHeader(); err != nil {
			t.Fatalf("AuthorizationHeader() error = %v", err)
		}
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("Expected 1 token request for consecutive calls within the clamped margin, got %d", got)
	}
}

// invalidatingCreds is a fake credential that switches to a fresh token when
// the client invalidates it after a 401.
type invalidatingCreds struct {
	header      string
	invalidated int
}

func (f *invalidatingCreds) AuthorizationHeader() (string, error) { return f.header, nil }
func (f *invalidatingCreds) Invalidate() {
	f.invalidated++
	f.header = "Bearer fresh"
}

func TestClient_401InvalidatesTokenAndRetriesOnce(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fresh" {
			t.Errorf("retry did not use the refreshed token: %s", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"article": {}}`))
	}))
	defer server.Close()

	creds := &invalidatingCreds{header: "Bearer stale"}
	client := &clientImpl{subdomain: "test", creds: creds, baseURLOverride: server.URL}

	if _, err := client.ShowArticle("en_us", 123); err != nil {
		t.Fatalf("ShowArticle() error = %v", err)
	}
	if creds.invalidated != 1 {
		t.Errorf("Invalidate called %d times, want 1", creds.invalidated)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("Expected 2 requests (401 then retry), got %d", got)
	}
}

func TestClient_401WithoutInvalidatorFailsImmediately(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := &clientImpl{
		subdomain:       "test",
		creds:           &TokenCredentials{Email: "user@example.com/token", Token: "t"},
		baseURLOverride: server.URL,
	}

	_, err := client.ShowArticle("en_us", 123)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Expected 401 error, got: %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("Expected 1 request (no retry for Basic auth), got %d", got)
	}
}

// TestClient_401RefreshesOAuthTokenAndRetries exercises the whole self-healing
// flow with a real OAuthCredentials: a token that is valid locally but revoked
// server-side gets a 401, is invalidated, refreshed, and the request is retried.
func TestClient_401RefreshesOAuthTokenAndRetries(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "fresh",
			"refresh_token": "newrefresh",
			"token_type": "bearer",
			"expires_in": 1800
		}`))
	}))
	defer tokenServer.Close()

	var apiRequests atomic.Int32
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiRequests.Add(1) == 1 {
			if got := r.Header.Get("Authorization"); got != "Bearer stale" {
				t.Errorf("first request Authorization = %s, want Bearer stale", got)
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fresh" {
			t.Errorf("retry Authorization = %s, want Bearer fresh", got)
		}
		_, _ = w.Write([]byte(`{"article": {}}`))
	}))
	defer apiServer.Close()

	creds := &OAuthCredentials{
		AccessToken:  "stale",
		RefreshToken: "refreshtoken",
		Expiry:       time.Now().Add(1 * time.Hour), // locally valid, revoked server-side
		ClientID:     "myclient",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: tokenServer.URL},
	}
	client := &clientImpl{subdomain: "test", creds: creds, baseURLOverride: apiServer.URL}

	if _, err := client.ShowArticle("en_us", 123); err != nil {
		t.Fatalf("ShowArticle() error = %v", err)
	}
	if got := apiRequests.Load(); got != 2 {
		t.Errorf("Expected 2 API requests (401 then retry), got %d", got)
	}
	if creds.AccessToken != "fresh" || creds.RefreshToken != "newrefresh" {
		t.Errorf("credentials after retry = (%s, %s), want (fresh, newrefresh)", creds.AccessToken, creds.RefreshToken)
	}
}

func TestClientCredentialsProvider_InvalidateForcesRefetch(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "citoken",
			"token_type": "bearer",
			"expires_in": 1800
		}`))
	}))
	defer server.Close()

	provider := &ClientCredentialsProvider{
		ClientID:     "ciclient",
		ClientSecret: "cisecret",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	if _, err := provider.AuthorizationHeader(); err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	provider.Invalidate()
	if _, err := provider.AuthorizationHeader(); err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("Expected 2 token requests after Invalidate, got %d", got)
	}
}

func TestClientCredentialsProvider_TokenRequestFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid_client"}`))
	}))
	defer server.Close()

	provider := &ClientCredentialsProvider{
		ClientID:     "ciclient",
		ClientSecret: "wrongsecret",
		OAuth:        &OAuthClient{subdomain: "test", baseURLOverride: server.URL},
	}

	_, err := provider.AuthorizationHeader()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Errorf("Expected error containing 'status 401', got: %v", err)
	}
}
