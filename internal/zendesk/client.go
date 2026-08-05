package zendesk

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "github.com/tukaelu/zgsync/internal/zendesk/httplog"
)

const (
	BaseURL = "https://%s.zendesk.com"
)

// httpClient is shared by API and OAuth token requests so a black-holed
// endpoint fails instead of blocking the CLI forever.
var httpClient = &http.Client{Timeout: 30 * time.Second}

func resolveBaseURL(subdomain, override string) string {
	if override != "" {
		return override
	}
	return fmt.Sprintf(BaseURL, subdomain)
}

type Client interface {
	CreateArticle(locale string, sectionID int, payload string) (string, error)
	UpdateArticle(locale string, articleID int, payload string) (string, error)
	ShowArticle(locale string, articleID int) (string, error)
	ArchiveArticle(articleID int) error
	CreateTranslation(articleID int, payload string) (string, error)
	UpdateTranslation(articleID int, locale string, payload string) (string, error)
	ShowTranslation(articleID int, locale string) (string, error)
}

type clientImpl struct {
	subdomain       string
	creds           Credentials
	baseURLOverride string
}

func NewClient(subdomain, email, token string) Client {
	return NewClientWithCredentials(subdomain, &TokenCredentials{Email: email, Token: token})
}

func NewClientWithCredentials(subdomain string, creds Credentials) Client {
	return &clientImpl{
		subdomain: subdomain,
		creds:     creds,
	}
}

// ClientBaseURL returns the effective base URL of the given Client.
// It panics if c is not a *clientImpl.
func ClientBaseURL(c Client) string {
	return c.(*clientImpl).baseURL()
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/#create-article
func (c *clientImpl) CreateArticle(locale string, sectionID int, payload string) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/%s/sections/%d/articles.json",
		locale,
		sectionID,
	)
	_payload := strings.NewReader(payload)
	return c.doRequest(http.MethodPost, endpoint, _payload)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/#update-article
func (c *clientImpl) UpdateArticle(locale string, articleID int, payload string) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/%s/articles/%d",
		locale,
		articleID,
	)
	_payload := strings.NewReader(payload)
	return c.doRequest(http.MethodPut, endpoint, _payload)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/#show-article
func (c *clientImpl) ShowArticle(locale string, articleID int) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/%s/articles/%d",
		locale,
		articleID,
	)
	return c.doRequest(http.MethodGet, endpoint, nil)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/#archive-article
func (c *clientImpl) ArchiveArticle(articleID int) error {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/articles/%d",
		articleID,
	)
	return c.doDeleteRequest(endpoint)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/#create-translation
func (c *clientImpl) CreateTranslation(articleID int, payload string) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/articles/%d/translations",
		articleID,
	)
	_payload := strings.NewReader(payload)
	return c.doRequest(http.MethodPost, endpoint, _payload)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/#update-translation
func (c *clientImpl) UpdateTranslation(articleID int, locale string, payload string) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/articles/%d/translations/%s",
		articleID,
		locale,
	)
	_payload := strings.NewReader(payload)
	return c.doRequest(http.MethodPut, endpoint, _payload)
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/#show-translation
func (c *clientImpl) ShowTranslation(articleID int, locale string) (string, error) {
	endpoint := fmt.Sprintf(
		"/api/v2/help_center/articles/%d/translations/%s",
		articleID,
		locale,
	)
	return c.doRequest(http.MethodGet, endpoint, nil)
}

func (c *clientImpl) doRequest(method string, endpoint string, payload io.Reader) (string, error) {
	return c.do(method, endpoint, payload, http.StatusOK, http.StatusCreated)
}

func (c *clientImpl) doDeleteRequest(endpoint string) error {
	_, err := c.do(http.MethodDelete, endpoint, nil, http.StatusNoContent)
	return err
}

func (c *clientImpl) do(method, endpoint string, payload io.Reader, okStatuses ...int) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	var body []byte
	if payload != nil {
		var err error
		if body, err = io.ReadAll(payload); err != nil {
			return "", err
		}
	}

	status, resBody, err := c.attempt(method, endpoint, body)
	if err != nil {
		return "", err
	}
	// A 401 despite a locally valid token means it was revoked or expired
	// server-side; discard the cached token and retry once so the OAuth
	// refresh/token flow can recover.
	if status == http.StatusUnauthorized {
		if inv, ok := c.creds.(interface{ Invalidate() }); ok {
			inv.Invalidate()
			if status, resBody, err = c.attempt(method, endpoint, body); err != nil {
				return "", err
			}
		}
	}

	for _, ok := range okStatuses {
		if status == ok {
			return string(resBody), nil
		}
	}
	return "", fmt.Errorf("unexpected status code: %d", status)
}

func (c *clientImpl) attempt(method, endpoint string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(method, c.baseURL()+endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}

	authz, err := c.creds.AuthorizationHeader()
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authz)

	res, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = res.Body.Close() }()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, err
	}
	return res.StatusCode, resBody, nil
}

func (c *clientImpl) baseURL() string {
	return resolveBaseURL(c.subdomain, c.baseURLOverride)
}

