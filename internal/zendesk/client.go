package zendesk

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	_ "github.com/tukaelu/zgsync/internal/zendesk/httplog"
)

const (
	BaseURL = "https://%s.zendesk.com"
)

type Client interface {
	CreateArticle(locale string, sectionID int, payload string) (string, error)
	UpdateArticle(locale string, articleID int, payload string) (string, error)
	ShowArticle(locale string, articleID int) (string, error)
	CreateTranslation(articleID int, payload string) (string, error)
	UpdateTranslation(articleID int, locale string, payload string) (string, error)
	ShowTranslation(articleID int, locale string) (string, error)
}

type clientImpl struct {
	subdomain string
	email     string
	token     string
}

func NewClient(subdomain, email, token string) Client {
	return &clientImpl{
		subdomain: subdomain,
		email:     email,
		token:     token,
	}
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
	if endpoint == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	reqURL := c.baseURL() + endpoint
	req, err := http.NewRequest(method, reqURL, payload)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.authorizationToken())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	resPayload, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(resPayload), nil
}

func (c *clientImpl) baseURL() string {
	return fmt.Sprintf(BaseURL, c.subdomain)
}

func (c *clientImpl) authorizationToken() string {
	return base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))
}
