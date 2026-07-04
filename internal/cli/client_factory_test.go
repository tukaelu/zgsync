package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

func TestGlobal_NewZendeskClient_AuthTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    Config
		expectErr string
	}{
		{
			name:   "token auth",
			config: Config{Subdomain: "test", AuthType: AuthTypeToken, Email: "user@example.com/token", Token: "token123"},
		},
		{
			name:   "empty auth_type falls back to token",
			config: Config{Subdomain: "test", Email: "user@example.com/token", Token: "token123"},
		},
		{
			name:   "client_credentials auth",
			config: Config{Subdomain: "test", AuthType: AuthTypeClientCredentials, OAuthClientID: "ci", OAuthClientSecret: "secret", OAuthScope: DefaultOAuthScope},
		},
		{
			name:      "unsupported auth_type",
			config:    Config{Subdomain: "test", AuthType: "bogus"},
			expectErr: "unsupported auth_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := &Global{Config: tt.config}
			client, err := g.NewZendeskClient()

			if tt.expectErr != "" {
				if err == nil {
					t.Fatalf("NewZendeskClient() = nil error, want error containing %q", tt.expectErr)
				}
				if !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("NewZendeskClient() error = %v, want error containing %q", err, tt.expectErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewZendeskClient() error = %v", err)
			}
			if got := zendesk.ClientBaseURL(client); got != "https://test.zendesk.com" {
				t.Errorf("ClientBaseURL() = %s, want https://test.zendesk.com", got)
			}
		})
	}
}

// setTempHome points the credential store at a temporary home directory and
// returns a store bound to the credentials.json inside it.
func setTempHome(t *testing.T) *CredentialStore {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)        // unix
	t.Setenv("USERPROFILE", home) // windows
	return NewCredentialStoreWithPath(filepath.Join(home, ".config", "zgsync", "credentials.json"))
}

func TestGlobal_NewZendeskClient_OAuth(t *testing.T) {
	store := setTempHome(t)
	if err := store.Save("test", &StoredCredential{ClientID: "myclient", AccessToken: "access123"}); err != nil {
		t.Fatal(err)
	}

	g := &Global{Config: Config{Subdomain: "test", AuthType: AuthTypeOAuth, OAuthClientID: "myclient"}}
	client, err := g.NewZendeskClient()
	if err != nil {
		t.Fatalf("NewZendeskClient() error = %v", err)
	}
	if got := zendesk.ClientBaseURL(client); got != "https://test.zendesk.com" {
		t.Errorf("ClientBaseURL() = %s, want https://test.zendesk.com", got)
	}
}

func TestGlobal_NewZendeskClient_OAuthWithoutLogin(t *testing.T) {
	setTempHome(t)

	g := &Global{Config: Config{Subdomain: "test", AuthType: AuthTypeOAuth, OAuthClientID: "myclient"}}
	_, err := g.NewZendeskClient()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected error to suggest `auth login`, got: %v", err)
	}
}

func TestNewOAuthCredentials_FromStore(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(t.TempDir() + "/credentials.json")
	if err := store.Save("test", &StoredCredential{
		ClientID:     "storedclient",
		AccessToken:  "access123",
		RefreshToken: "refresh456",
	}); err != nil {
		t.Fatal(err)
	}

	creds, err := newOAuthCredentials(Config{Subdomain: "test"}, store)
	if err != nil {
		t.Fatalf("newOAuthCredentials() error = %v", err)
	}

	if creds.ClientID != "storedclient" {
		t.Errorf("ClientID = %s, want storedclient (from store)", creds.ClientID)
	}
	authz, err := creds.AuthorizationHeader()
	if err != nil {
		t.Fatalf("AuthorizationHeader() error = %v", err)
	}
	if authz != "Bearer access123" {
		t.Errorf("AuthorizationHeader() = %s, want Bearer access123", authz)
	}
}

func TestNewOAuthCredentials_ClientIDMismatchIsRejected(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(t.TempDir() + "/credentials.json")
	if err := store.Save("test", &StoredCredential{ClientID: "storedclient", AccessToken: "a"}); err != nil {
		t.Fatal(err)
	}

	_, err := newOAuthCredentials(Config{Subdomain: "test", OAuthClientID: "configclient"}, store)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "does not match") || !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected client ID mismatch error suggesting `auth login`, got: %v", err)
	}
}

func TestNewOAuthCredentials_MatchingClientIDIsAccepted(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(t.TempDir() + "/credentials.json")
	if err := store.Save("test", &StoredCredential{ClientID: "myclient", AccessToken: "a"}); err != nil {
		t.Fatal(err)
	}

	creds, err := newOAuthCredentials(Config{Subdomain: "test", OAuthClientID: "myclient"}, store)
	if err != nil {
		t.Fatalf("newOAuthCredentials() error = %v", err)
	}
	if creds.ClientID != "myclient" {
		t.Errorf("ClientID = %s, want myclient", creds.ClientID)
	}
}

func TestNewOAuthCredentials_OnRefreshPersistsTokens(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(filepath.Join(t.TempDir(), "credentials.json"))
	if err := store.Save("test", &StoredCredential{
		ClientID:     "myclient",
		AccessToken:  "old",
		RefreshToken: "oldrefresh",
		Scope:        "hc:read",
	}); err != nil {
		t.Fatal(err)
	}

	creds, err := newOAuthCredentials(Config{Subdomain: "test"}, store)
	if err != nil {
		t.Fatalf("newOAuthCredentials() error = %v", err)
	}

	// Simulate a login performed while this process is running: OnRefresh
	// must re-load the file and keep this newer state instead of writing
	// back the startup-time snapshot.
	if err := store.Save("test", &StoredCredential{
		ClientID:     "myclient",
		AccessToken:  "old",
		RefreshToken: "oldrefresh",
		Scope:        "hc:read hc:write",
	}); err != nil {
		t.Fatal(err)
	}

	expiry := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	if err := creds.OnRefresh("newaccess", "newrefresh", expiry); err != nil {
		t.Fatalf("OnRefresh() error = %v", err)
	}

	saved, err := store.Load("test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if saved.AccessToken != "newaccess" || saved.RefreshToken != "newrefresh" {
		t.Errorf("saved tokens = (%s, %s), want (newaccess, newrefresh)", saved.AccessToken, saved.RefreshToken)
	}
	if !saved.Expiry.Equal(expiry) {
		t.Errorf("saved expiry = %v, want %v", saved.Expiry, expiry)
	}
	if saved.Scope != "hc:read hc:write" {
		t.Errorf("saved scope = %s, want hc:read hc:write (the re-loaded state must be kept)", saved.Scope)
	}
}

func TestNewOAuthCredentials_OnRefreshSaveFailureIsNotFatal(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not applicable on Windows")
	}

	dir := t.TempDir()
	store := NewCredentialStoreWithPath(filepath.Join(dir, "credentials.json"))
	if err := store.Save("test", &StoredCredential{ClientID: "myclient", AccessToken: "old", RefreshToken: "oldrefresh"}); err != nil {
		t.Fatal(err)
	}

	creds, err := newOAuthCredentials(Config{Subdomain: "test"}, store)
	if err != nil {
		t.Fatalf("newOAuthCredentials() error = %v", err)
	}

	// Make the directory read-only so the atomic write fails.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	// Zendesk has already rotated the refresh token at this point; failing
	// would discard a valid access token, so OnRefresh must warn and return nil.
	if err := creds.OnRefresh("newaccess", "newrefresh", time.Now().Add(30*time.Minute)); err != nil {
		t.Errorf("OnRefresh() = %v, want nil despite the save failure", err)
	}
}

func TestNewOAuthCredentials_MissingStoredCredential(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(t.TempDir() + "/credentials.json")

	_, err := newOAuthCredentials(Config{Subdomain: "test"}, store)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected error to suggest `auth login`, got: %v", err)
	}
}
