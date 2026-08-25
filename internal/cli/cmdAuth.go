package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandAuth struct {
	Login CommandAuthLogin `cmd:"login" help:"Log in to Zendesk via OAuth (authorization code grant with PKCE) and save the tokens locally."`
}

type CommandAuthLogin struct {
	Port int    `name:"port" help:"Port of the local callback server. The OAuth client must register http://localhost:<port>/callback as a redirect URL." default:"8976"`
	Bind string `name:"bind" help:"Bind address of the local callback server (e.g. 0.0.0.0 for Docker)." default:"localhost"`

	oauthClient *zendesk.OAuthClient
	credStore   *CredentialStore
	openBrowser func(string) error
	timeout     time.Duration
}

func (c *CommandAuthLogin) AfterApply(g *Global) error {
	c.oauthClient = zendesk.NewOAuthClient(g.Config.Subdomain)
	c.credStore = NewCredentialStore(g.AbsConfig())
	c.openBrowser = openBrowser
	c.timeout = 5 * time.Minute
	return nil
}

type callbackResult struct {
	code string
	err  error
}

func (c *CommandAuthLogin) Run(g *Global) error {
	clientID := g.Config.OAuthClientID
	if clientID == "" {
		return fmt.Errorf("oauth_client_id is not set in %s", g.AbsConfig())
	}

	verifier, err := randomURLSafeString(32)
	if err != nil {
		return err
	}
	state, err := randomURLSafeString(16)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(c.Bind, strconv.Itoa(c.Port)))
	if err != nil {
		return fmt.Errorf("failed to start the callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	resultCh := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// Ignore requests that are not an OAuth response (prefetch, port scan,
		// stale tab) and requests with a wrong state, so a stray hit cannot
		// consume the result slot before the genuine callback arrives.
		if q.Get("code") == "" && q.Get("error") == "" {
			http.NotFound(w, r)
			return
		}
		if q.Get("state") != state {
			http.Error(w, "State mismatch. Retry the login from the terminal.", http.StatusBadRequest)
			return
		}

		result := callbackResult{}
		if errParam := q.Get("error"); errParam != "" {
			result.err = fmt.Errorf("authorization was denied: %s (%s)", errParam, q.Get("error_description"))
		} else {
			result.code = q.Get("code")
		}

		if result.err != nil {
			http.Error(w, "Authorization failed. You may close this window and check the terminal.", http.StatusBadRequest)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprint(w, "<html><body>Authorized. You may close this window and return to the terminal.</body></html>")
		}

		select {
		case resultCh <- result:
		default:
		}
	})

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	authorizeURL := buildAuthorizeURL(fmt.Sprintf(zendesk.BaseURL, g.Config.Subdomain), url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {g.Config.OAuthScope},
		"state":                 {state},
		"code_challenge":        {codeChallengeS256(verifier)},
		"code_challenge_method": {"S256"},
	})

	fmt.Printf("Opening the browser to authorize zgsync. If it does not open, visit:\n\n  %s\n\n", authorizeURL)
	if err := c.openBrowser(authorizeURL); err != nil {
		fmt.Printf("Failed to open the browser automatically: %v\n", err)
	}

	var result callbackResult
	select {
	case result = <-resultCh:
	case <-time.After(c.timeout):
		return fmt.Errorf("timed out waiting for the authorization callback; make sure %s is registered as a redirect URL of the OAuth client", redirectURI)
	}
	if result.err != nil {
		return result.err
	}

	token, err := c.oauthClient.ExchangeAuthorizationCode(clientID, result.code, redirectURI, verifier)
	if err != nil {
		return err
	}

	cred := &StoredCredential{
		ClientID:     clientID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.ExpiryTime(),
		Scope:        token.Scope,
	}
	if err := c.credStore.Save(g.Config.Subdomain, cred); err != nil {
		return err
	}

	fmt.Printf("Logged in to %s.zendesk.com. Tokens were saved to %s\n", g.Config.Subdomain, c.credStore.Path())
	return nil
}

func buildAuthorizeURL(baseURL string, params url.Values) string {
	return baseURL + "/oauth/authorizations/new?" + params.Encode()
}

func randomURLSafeString(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}
