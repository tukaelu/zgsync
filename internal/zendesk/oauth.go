package zendesk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OAuthTokenResponse is the response body of POST /oauth/tokens.
// refs: https://developer.zendesk.com/api-reference/ticketing/oauth/oauth_tokens/
type OAuthTokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
}

// ExpiryTime converts ExpiresIn into an absolute expiry with a safety margin
// applied, clamped to half the lifetime so short-lived tokens remain usable
// after being issued. A zero return value means the token does not expire.
func (r *OAuthTokenResponse) ExpiryTime() time.Time {
	if r.ExpiresIn <= 0 {
		return time.Time{}
	}
	lifetime := time.Duration(r.ExpiresIn) * time.Second
	margin := tokenExpiryMargin
	if half := lifetime / 2; half < margin {
		margin = half
	}
	return time.Now().Add(lifetime - margin)
}

// OAuthClient issues token requests against the Zendesk OAuth token endpoint.
type OAuthClient struct {
	subdomain       string
	baseURLOverride string
}

func NewOAuthClient(subdomain string) *OAuthClient {
	return &OAuthClient{subdomain: subdomain}
}

// NewOAuthClientWithBaseURL creates an OAuthClient that sends requests to baseURL
// instead of the Zendesk endpoint. Intended for tests.
func NewOAuthClientWithBaseURL(subdomain, baseURL string) *OAuthClient {
	return &OAuthClient{subdomain: subdomain, baseURLOverride: baseURL}
}

// ExchangeAuthorizationCode exchanges an authorization code for tokens using PKCE.
// refs: https://developer.zendesk.com/api-reference/ticketing/oauth/grant_type_tokens/#creating-authorization-code-grant-type-tokens
func (o *OAuthClient) ExchangeAuthorizationCode(clientID, code, redirectURI, codeVerifier string) (*OAuthTokenResponse, error) {
	return o.doTokenRequest(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
		"code_verifier": codeVerifier,
	})
}

// RefreshToken obtains a new access token using a refresh token.
// refs: https://developer.zendesk.com/api-reference/ticketing/oauth/grant_type_tokens/#creating-refresh-token-grant-type-tokens
func (o *OAuthClient) RefreshToken(clientID, refreshToken string) (*OAuthTokenResponse, error) {
	return o.doTokenRequest(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     clientID,
	})
}

// ClientCredentials obtains an access token using the client_credentials grant.
// The token user is the owner of the OAuth client.
// refs: https://developer.zendesk.com/api-reference/ticketing/oauth/grant_type_tokens/#creating-client-credentials-grant-type-tokens
func (o *OAuthClient) ClientCredentials(clientID, clientSecret, scope string) (*OAuthTokenResponse, error) {
	return o.doTokenRequest(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"scope":         scope,
	})
}

func (o *OAuthClient) doTokenRequest(params map[string]string) (*OAuthTokenResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	reqURL := o.baseURL() + "/oauth/tokens"
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("token request failed with status %d: %s", res.StatusCode, string(resBody))
	}

	token := &OAuthTokenResponse{}
	if err := json.Unmarshal(resBody, token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("token response does not contain an access token")
	}
	return token, nil
}

func (o *OAuthClient) baseURL() string {
	return resolveBaseURL(o.subdomain, o.baseURLOverride)
}
