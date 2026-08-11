package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

// NewZendeskClient builds a Zendesk API client with credentials matching auth_type.
func (g *Global) NewZendeskClient() (zendesk.Client, error) {
	creds, err := g.newCredentials()
	if err != nil {
		return nil, err
	}
	return zendesk.NewClientWithCredentials(g.Config.Subdomain, creds), nil
}

func (g *Global) newCredentials() (zendesk.Credentials, error) {
	switch g.Config.AuthType {
	case "", AuthTypeToken:
		return &zendesk.TokenCredentials{Email: g.Config.Email, Token: g.Config.Token}, nil
	case AuthTypeOAuth:
		return newOAuthCredentials(g.Config, NewCredentialStore(g.AbsConfig()))
	case AuthTypeClientCredentials:
		return &zendesk.ClientCredentialsProvider{
			ClientID:     g.Config.OAuthClientID,
			ClientSecret: g.Config.OAuthClientSecret,
			Scope:        g.Config.OAuthScope,
			OAuth:        zendesk.NewOAuthClient(g.Config.Subdomain),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth_type: %s", g.Config.AuthType)
	}
}

func newOAuthCredentials(cfg Config, store *CredentialStore) (*zendesk.OAuthCredentials, error) {
	cred, err := store.Load(cfg.Subdomain)
	if err != nil {
		return nil, err
	}
	// The refresh token is bound to the client that issued it, so a config
	// client ID that differs from the stored one can never refresh successfully.
	if cfg.OAuthClientID != "" && cred.ClientID != "" && cfg.OAuthClientID != cred.ClientID {
		return nil, fmt.Errorf(
			"oauth_client_id %q does not match the client %q the saved tokens were issued to; run `zgsync auth login` again",
			cfg.OAuthClientID, cred.ClientID,
		)
	}
	clientID := cred.ClientID
	if clientID == "" {
		clientID = cfg.OAuthClientID
	}
	return &zendesk.OAuthCredentials{
		AccessToken:  cred.AccessToken,
		RefreshToken: cred.RefreshToken,
		Expiry:       cred.Expiry,
		ClientID:     clientID,
		OAuth:        zendesk.NewOAuthClient(cfg.Subdomain),
		OnRefresh: func(accessToken, refreshToken string, expiry time.Time) error {
			// Re-load so a login performed while this process was running
			// is not clobbered with the startup-time snapshot.
			latest, err := store.Load(cfg.Subdomain)
			if err != nil {
				latest = cred
			}
			latest.AccessToken = accessToken
			latest.RefreshToken = refreshToken
			latest.Expiry = expiry
			if err := store.Save(cfg.Subdomain, latest); err != nil {
				// Zendesk already rotated the refresh token; failing here would
				// discard a valid access token, so warn and keep going.
				fmt.Fprintf(os.Stderr,
					"warning: failed to save refreshed tokens to %s: %v\nif the next run fails to authenticate, run `zgsync auth login` again\n",
					store.Path(), err,
				)
			}
			return nil
		},
	}, nil
}
