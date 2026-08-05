package zendesk

import (
	"encoding/base64"
	"fmt"
	"time"
)

// tokenExpiryMargin is subtracted from the token lifetime (see
// OAuthTokenResponse.ExpiryTime) so that a token close to expiration is not
// used for a request that may outlive it.
const tokenExpiryMargin = 60 * time.Second

// tokenExpired reports whether a token with the given expiry can no longer be
// used. A zero expiry means the token does not expire.
func tokenExpired(expiry time.Time) bool {
	return !expiry.IsZero() && !time.Now().Before(expiry)
}

// Credentials provides the value of the Authorization header for API requests.
type Credentials interface {
	AuthorizationHeader() (string, error)
}

// TokenCredentials authenticates with a Zendesk API token using Basic authentication.
// refs: https://developer.zendesk.com/api-reference/introduction/security-and-auth/#api-token
type TokenCredentials struct {
	Email string
	Token string
}

func (c *TokenCredentials) AuthorizationHeader() (string, error) {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.Email+":"+c.Token)), nil
}

// OAuthCredentials authenticates with an OAuth access token obtained via the
// authorization_code grant, refreshing it automatically when expired.
type OAuthCredentials struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	ClientID     string
	OAuth        *OAuthClient
	// OnRefresh is called after a successful refresh so the caller can persist the new tokens.
	OnRefresh func(accessToken, refreshToken string, expiry time.Time) error
}

func (c *OAuthCredentials) AuthorizationHeader() (string, error) {
	if c.AccessToken == "" || tokenExpired(c.Expiry) {
		if err := c.refresh(); err != nil {
			return "", err
		}
	}
	return "Bearer " + c.AccessToken, nil
}

// Invalidate discards the cached access token so the next request refreshes it.
// The API client calls this when a request comes back 401, which means the
// token was revoked or expired server-side despite a valid local expiry.
func (c *OAuthCredentials) Invalidate() {
	c.AccessToken = ""
}

func (c *OAuthCredentials) refresh() error {
	if c.RefreshToken == "" {
		return fmt.Errorf("the access token is expired and no refresh token is available; run `zgsync auth login` again")
	}
	res, err := c.OAuth.RefreshToken(c.ClientID, c.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh the access token; run `zgsync auth login` again: %w", err)
	}
	c.AccessToken = res.AccessToken
	if res.RefreshToken != "" {
		c.RefreshToken = res.RefreshToken
	}
	c.Expiry = res.ExpiryTime()
	if c.OnRefresh != nil {
		if err := c.OnRefresh(c.AccessToken, c.RefreshToken, c.Expiry); err != nil {
			return err
		}
	}
	return nil
}

// ClientCredentialsProvider obtains an access token via the client_credentials
// grant on demand and caches it in memory for the lifetime of the process.
// Intended for non-interactive environments such as CI.
type ClientCredentialsProvider struct {
	ClientID     string
	ClientSecret string
	Scope        string
	OAuth        *OAuthClient

	accessToken string
	expiry      time.Time
}

func (c *ClientCredentialsProvider) AuthorizationHeader() (string, error) {
	if c.accessToken == "" || tokenExpired(c.expiry) {
		res, err := c.OAuth.ClientCredentials(c.ClientID, c.ClientSecret, c.Scope)
		if err != nil {
			return "", err
		}
		c.accessToken = res.AccessToken
		c.expiry = res.ExpiryTime()
	}
	return "Bearer " + c.accessToken, nil
}

// Invalidate discards the cached access token so the next request fetches a new one.
func (c *ClientCredentialsProvider) Invalidate() {
	c.accessToken = ""
}
