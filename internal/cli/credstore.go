package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StoredCredential holds the OAuth tokens saved by `zgsync auth login`.
type StoredCredential struct {
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

// CredentialStore reads and writes OAuth tokens in a JSON file keyed by subdomain.
// The file is managed exclusively by zgsync and kept separate from config.yaml
// so that tokens never end up in a user-edited (and possibly committed) file.
type CredentialStore struct {
	path string
}

func NewCredentialStore() (*CredentialStore, error) {
	dir, err := zgsyncConfigDir()
	if err != nil {
		return nil, err
	}
	return &CredentialStore{
		path: filepath.Join(dir, "credentials.json"),
	}, nil
}

func NewCredentialStoreWithPath(path string) *CredentialStore {
	return &CredentialStore{path: path}
}

func (s *CredentialStore) Path() string {
	return s.path
}

func (s *CredentialStore) Load(subdomain string) (*StoredCredential, error) {
	creds, err := s.loadAll()
	if err != nil {
		return nil, err
	}
	cred, ok := creds[subdomain]
	if !ok {
		return nil, fmt.Errorf("no credentials found for subdomain %q in %s; run `zgsync auth login` first", subdomain, s.path)
	}
	return &cred, nil
}

func (s *CredentialStore) Save(subdomain string, cred *StoredCredential) error {
	creds, err := s.loadAll()
	if err != nil {
		return err
	}
	creds[subdomain] = *cred

	b, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return s.writeAtomic(b)
}

// writeAtomic writes via a temp file and rename so a crash mid-write can
// never leave a truncated credentials.json that would brick every subdomain.
func (s *CredentialStore) writeAtomic(b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".credentials-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

func (s *CredentialStore) loadAll() (map[string]StoredCredential, error) {
	creds := map[string]StoredCredential{}
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return creds, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", s.path, err)
	}
	return creds, nil
}
