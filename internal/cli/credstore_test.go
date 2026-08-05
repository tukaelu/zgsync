package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCredentialStore_SaveWithBrokenFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte("{broken json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewCredentialStoreWithPath(path)

	err := store.Save("test", &StoredCredential{AccessToken: "a"})
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestCredentialStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(filepath.Join(t.TempDir(), "credentials.json"))

	cred := &StoredCredential{
		ClientID:     "myclient",
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		Expiry:       time.Now().Add(30 * time.Minute).Truncate(time.Second),
		Scope:        "hc:read hc:write",
	}
	if err := store.Save("mycompany", cred); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load("mycompany")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ClientID != cred.ClientID ||
		loaded.AccessToken != cred.AccessToken ||
		loaded.RefreshToken != cred.RefreshToken ||
		loaded.Scope != cred.Scope ||
		!loaded.Expiry.Equal(cred.Expiry) {
		t.Errorf("Load() = %+v, want %+v", loaded, cred)
	}
}

func TestCredentialStore_SavePreservesOtherSubdomains(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(filepath.Join(t.TempDir(), "credentials.json"))

	if err := store.Save("company-a", &StoredCredential{ClientID: "a", AccessToken: "tokena"}); err != nil {
		t.Fatalf("Save(company-a) error = %v", err)
	}
	if err := store.Save("company-b", &StoredCredential{ClientID: "b", AccessToken: "tokenb"}); err != nil {
		t.Fatalf("Save(company-b) error = %v", err)
	}

	credA, err := store.Load("company-a")
	if err != nil {
		t.Fatalf("Load(company-a) error = %v", err)
	}
	if credA.AccessToken != "tokena" {
		t.Errorf("Load(company-a).AccessToken = %s, want tokena", credA.AccessToken)
	}
}

func TestCredentialStore_LoadMissingSubdomain(t *testing.T) {
	t.Parallel()

	store := NewCredentialStoreWithPath(filepath.Join(t.TempDir(), "credentials.json"))

	_, err := store.Load("unknown")
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("Expected error to suggest `auth login`, got: %v", err)
	}
}

func TestCredentialStore_FilePermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not applicable on Windows")
	}

	path := filepath.Join(t.TempDir(), "nested", "credentials.json")
	store := NewCredentialStoreWithPath(path)

	if err := store.Save("mycompany", &StoredCredential{ClientID: "c", AccessToken: "t"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials.json permission = %o, want 600", perm)
	}
}

func TestCredentialStore_LoadBrokenFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte("{broken json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewCredentialStoreWithPath(path)

	_, err := store.Load("mycompany")
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}
