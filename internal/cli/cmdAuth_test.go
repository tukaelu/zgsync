package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

// browserStub simulates the Zendesk authorization page: it parses the authorize
// URL and redirects back to the local callback server like a real authorization would.
func browserStub(t *testing.T, mutate func(q url.Values)) func(string) error {
	t.Helper()
	return func(authorizeURL string) error {
		u, err := url.Parse(authorizeURL)
		if err != nil {
			return err
		}
		q := u.Query()

		callback := q.Get("redirect_uri") + "?" + url.Values{
			"code":  {"authcode123"},
			"state": {q.Get("state")},
		}.Encode()
		cu, err := url.Parse(callback)
		if err != nil {
			return err
		}
		cq := cu.Query()
		if mutate != nil {
			mutate(cq)
		}
		cu.RawQuery = cq.Encode()

		go func() {
			res, err := http.Get(cu.String())
			if err != nil {
				t.Errorf("callback request failed: %v", err)
				return
			}
			_ = res.Body.Close()
		}()
		return nil
	}
}

func newAuthLoginCommand(t *testing.T, tokenServerURL string, browser func(string) error) (*CommandAuthLogin, *CredentialStore) {
	t.Helper()
	store := NewCredentialStoreWithPath(filepath.Join(t.TempDir(), "credentials.json"))
	cmd := &CommandAuthLogin{
		Port:        0,
		Bind:        "localhost",
		oauthClient: zendesk.NewOAuthClientWithBaseURL("test", tokenServerURL),
		credStore:   store,
		openBrowser: browser,
		timeout:     5 * time.Second,
	}
	return cmd, store
}

func authLoginGlobal() *Global {
	return &Global{
		Config: Config{
			Subdomain:     "test",
			AuthType:      AuthTypeOAuth,
			OAuthClientID: "myclient",
			OAuthScope:    DefaultOAuthScope,
		},
	}
}

func TestCommandAuthLogin_Success(t *testing.T) {
	t.Parallel()

	var tokenRequest map[string]string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := map[string]string{}
		if err := jsonDecode(r, &params); err != nil {
			t.Errorf("failed to decode token request: %v", err)
		}
		tokenRequest = params
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "access123",
			"refresh_token": "refresh456",
			"token_type": "bearer",
			"scope": "hc:read hc:write",
			"expires_in": 1800
		}`))
	}))
	defer tokenServer.Close()

	cmd, store := newAuthLoginCommand(t, tokenServer.URL, browserStub(t, nil))

	if err := cmd.Run(authLoginGlobal()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if tokenRequest["grant_type"] != "authorization_code" {
		t.Errorf("grant_type = %s, want authorization_code", tokenRequest["grant_type"])
	}
	if tokenRequest["code"] != "authcode123" {
		t.Errorf("code = %s, want authcode123", tokenRequest["code"])
	}
	if tokenRequest["code_verifier"] == "" {
		t.Error("code_verifier is empty; PKCE is not applied")
	}

	cred, err := store.Load("test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cred.AccessToken != "access123" || cred.RefreshToken != "refresh456" || cred.ClientID != "myclient" {
		t.Errorf("stored credential = %+v", cred)
	}
	if cred.Expiry.IsZero() {
		t.Error("stored credential has zero expiry")
	}
}

func TestCommandAuthLogin_StateMismatchKeepsWaiting(t *testing.T) {
	t.Parallel()

	// A callback with a wrong state must not consume the result slot;
	// with no valid callback arriving, the login times out.
	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", browserStub(t, func(q url.Values) {
		q.Set("state", "tampered")
	}))
	cmd.timeout = 300 * time.Millisecond

	err := cmd.Run(authLoginGlobal())
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestCommandAuthLogin_StrayRequestDoesNotAbortLogin(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "access123", "token_type": "bearer", "expires_in": 1800}`))
	}))
	defer tokenServer.Close()

	// The browser stub first sends a paramless stray request to /callback,
	// then the genuine redirect; the login must still succeed.
	browser := func(authorizeURL string) error {
		u, err := url.Parse(authorizeURL)
		if err != nil {
			return err
		}
		q := u.Query()
		redirectURI := q.Get("redirect_uri")

		go func() {
			if res, err := http.Get(redirectURI); err == nil {
				_ = res.Body.Close()
			}
			cb := url.Values{"code": {"authcode123"}, "state": {q.Get("state")}}
			if res, err := http.Get(redirectURI + "?" + cb.Encode()); err == nil {
				_ = res.Body.Close()
			}
		}()
		return nil
	}

	cmd, store := newAuthLoginCommand(t, tokenServer.URL, browser)

	if err := cmd.Run(authLoginGlobal()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := store.Load("test"); err != nil {
		t.Errorf("credentials were not saved: %v", err)
	}
}

func TestCommandAuthLogin_AuthorizationDenied(t *testing.T) {
	t.Parallel()

	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", browserStub(t, func(q url.Values) {
		q.Del("code")
		q.Set("error", "access_denied")
		q.Set("error_description", "The user denied the request")
	}))

	err := cmd.Run(authLoginGlobal())
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("Expected access_denied error, got: %v", err)
	}
}

func TestCommandAuthLogin_MissingClientID(t *testing.T) {
	t.Parallel()

	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", browserStub(t, nil))

	g := authLoginGlobal()
	g.Config.OAuthClientID = ""

	err := cmd.Run(g)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "oauth_client_id") {
		t.Errorf("Expected oauth_client_id error, got: %v", err)
	}
}

func TestCommandAuthLogin_Timeout(t *testing.T) {
	t.Parallel()

	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", func(string) error {
		return nil // never hits the callback
	})
	cmd.timeout = 100 * time.Millisecond

	err := cmd.Run(authLoginGlobal())
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestCommandAuthLogin_AfterApply verifies that AfterApply wires up the oauth
// client, credential store, browser opener, and timeout correctly.
func TestCommandAuthLogin_AfterApply(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cmd := &CommandAuthLogin{}
	if err := cmd.AfterApply(authLoginGlobal()); err != nil {
		t.Fatalf("AfterApply() error = %v", err)
	}
	if cmd.oauthClient == nil || cmd.credStore == nil || cmd.openBrowser == nil {
		t.Error("AfterApply() did not wire up all fields")
	}
	if cmd.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", cmd.timeout)
	}
}

func TestCommandAuthLogin_BrowserOpenFailureIsNonFatal(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access123","token_type":"bearer","expires_in":1800}`))
	}))
	defer tokenServer.Close()

	// Returns an error, but still fires the callback so the login can succeed.
	browser := func(authorizeURL string) error {
		u, err := url.Parse(authorizeURL)
		if err != nil {
			return err
		}
		q := u.Query()
		cb := q.Get("redirect_uri") + "?" + url.Values{
			"code": {"authcode123"}, "state": {q.Get("state")},
		}.Encode()
		go func() {
			if res, err := http.Get(cb); err == nil {
				_ = res.Body.Close()
			}
		}()
		return fmt.Errorf("browser not found")
	}

	cmd, store := newAuthLoginCommand(t, tokenServer.URL, browser)
	if err := cmd.Run(authLoginGlobal()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := store.Load("test"); err != nil {
		t.Errorf("credentials were not saved: %v", err)
	}
}

func TestCommandAuthLogin_ListenPortInUse(t *testing.T) {
	t.Parallel()

	// Keep a listener alive on a known port so Run cannot bind to it.
	blocker, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Close() }()
	port := blocker.Addr().(*net.TCPAddr).Port

	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", func(string) error { return nil })
	cmd.Port = port

	err = cmd.Run(authLoginGlobal())
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "callback server") {
		t.Errorf("Expected callback server error, got: %v", err)
	}
}

func TestCommandAuthLogin_BindAllInterfaces(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access123","token_type":"bearer","expires_in":1800}`))
	}))
	defer tokenServer.Close()

	var capturedAuthorizeURL string
	browser := func(authorizeURL string) error {
		capturedAuthorizeURL = authorizeURL
		return browserStub(t, nil)(authorizeURL)
	}

	cmd, store := newAuthLoginCommand(t, tokenServer.URL, browser)
	cmd.Bind = "0.0.0.0"

	if err := cmd.Run(authLoginGlobal()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// redirect_uri must still use localhost regardless of bind address
	u, err := url.Parse(capturedAuthorizeURL)
	if err != nil {
		t.Fatalf("failed to parse authorize URL: %v", err)
	}
	redirectURI := u.Query().Get("redirect_uri")
	if !strings.HasPrefix(redirectURI, "http://localhost:") {
		t.Errorf("redirect_uri = %s, want http://localhost:... prefix", redirectURI)
	}

	if _, err := store.Load("test"); err != nil {
		t.Errorf("credentials were not saved: %v", err)
	}
}

func TestCommandAuthLogin_BindAllInterfacesPortInUse(t *testing.T) {
	t.Parallel()

	blocker, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Close() }()
	port := blocker.Addr().(*net.TCPAddr).Port

	cmd, _ := newAuthLoginCommand(t, "http://invalid.example.com", func(string) error { return nil })
	cmd.Bind = "0.0.0.0"
	cmd.Port = port

	err = cmd.Run(authLoginGlobal())
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "callback server") {
		t.Errorf("Expected callback server error, got: %v", err)
	}
}

func TestCodeChallengeS256(t *testing.T) {
	t.Parallel()

	// test vector from RFC 7636 appendix B
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := codeChallengeS256(verifier); got != expected {
		t.Errorf("codeChallengeS256() = %s, want %s", got, expected)
	}
}

func jsonDecode(r *http.Request, v interface{}) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
