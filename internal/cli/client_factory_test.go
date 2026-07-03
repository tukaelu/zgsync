package cli

import (
	"strings"
	"testing"

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
