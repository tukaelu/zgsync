package testhelper

import (
	"encoding/json"
)

// MockZendeskClient is a centralized mock implementation of zendesk.Client
type MockZendeskClient struct {
	CreateArticleFunc     func(locale string, sectionID int, payload string) (string, error)
	UpdateArticleFunc     func(locale string, articleID int, payload string) (string, error)
	ShowArticleFunc       func(locale string, articleID int) (string, error)
	CreateTranslationFunc func(articleID int, payload string) (string, error)
	UpdateTranslationFunc func(articleID int, locale string, payload string) (string, error)
	ShowTranslationFunc   func(articleID int, locale string) (string, error)
}

// Article represents a Zendesk article for mock responses
type Article struct {
	ID        int    `json:"id"`
	SectionID int    `json:"section_id"`
	Title     string `json:"title"`
	Locale    string `json:"locale,omitempty"`
}

// Translation represents a Zendesk translation for mock responses
type Translation struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Locale   string `json:"locale"`
	SourceID int    `json:"source_id"`
}

// CreateDefaultArticleResponse creates a properly formatted JSON response for articles
func CreateDefaultArticleResponse(id, sectionID int) string {
	article := Article{
		ID:        id,
		SectionID: sectionID,
		Title:     "Test Article",
	}
	response := map[string]Article{"article": article}
	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// CreateDefaultTranslationResponse creates a properly formatted JSON response for translations
func CreateDefaultTranslationResponse(id, sourceID int, locale string) string {
	translation := Translation{
		ID:       id,
		Title:    "Test Translation",
		Body:     "<h1>Test</h1>",
		Locale:   locale,
		SourceID: sourceID,
	}
	response := map[string]Translation{"translation": translation}
	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// CreateArticle implements zendesk.Client
func (m *MockZendeskClient) CreateArticle(locale string, sectionID int, payload string) (string, error) {
	if m.CreateArticleFunc != nil {
		return m.CreateArticleFunc(locale, sectionID, payload)
	}
	return CreateDefaultArticleResponse(123, sectionID), nil
}

// UpdateArticle implements zendesk.Client
func (m *MockZendeskClient) UpdateArticle(locale string, articleID int, payload string) (string, error) {
	if m.UpdateArticleFunc != nil {
		return m.UpdateArticleFunc(locale, articleID, payload)
	}
	return CreateDefaultArticleResponse(articleID, 456), nil
}

// ShowArticle implements zendesk.Client
func (m *MockZendeskClient) ShowArticle(locale string, articleID int) (string, error) {
	if m.ShowArticleFunc != nil {
		return m.ShowArticleFunc(locale, articleID)
	}
	return CreateDefaultArticleResponse(articleID, 456), nil
}

// CreateTranslation implements zendesk.Client
func (m *MockZendeskClient) CreateTranslation(articleID int, payload string) (string, error) {
	if m.CreateTranslationFunc != nil {
		return m.CreateTranslationFunc(articleID, payload)
	}
	return CreateDefaultTranslationResponse(1, articleID, "en_us"), nil
}

// UpdateTranslation implements zendesk.Client
func (m *MockZendeskClient) UpdateTranslation(articleID int, locale string, payload string) (string, error) {
	if m.UpdateTranslationFunc != nil {
		return m.UpdateTranslationFunc(articleID, locale, payload)
	}
	return CreateDefaultTranslationResponse(1, articleID, locale), nil
}

// ShowTranslation implements zendesk.Client
func (m *MockZendeskClient) ShowTranslation(articleID int, locale string) (string, error) {
	if m.ShowTranslationFunc != nil {
		return m.ShowTranslationFunc(articleID, locale)
	}
	return CreateDefaultTranslationResponse(1, articleID, locale), nil
}

// IntPtr is a helper function to create int pointers
func IntPtr(i int) *int {
	return &i
}