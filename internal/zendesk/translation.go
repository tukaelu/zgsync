package zendesk

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/#update-translation
type Translation struct {
	Body        string `json:"body,omitempty" yaml:"-"`
	CreatedAt   string `json:"created_at,omitempty" yaml:"created_at"`
	CreatedById int    `json:"created_by_id,omitempty" yaml:"created_by_id"`
	Draft       bool   `json:"draft,omitempty" yaml:"draft"`
	HtmlURL     string `json:"html_url,omitempty" yaml:"html_url"`
	ID          int    `json:"id" yaml:"id"`
	Locale      string `json:"locale" yaml:"locale"`
	Outdated    bool   `json:"outdated,omitempty" yaml:"outdated"`
	SourceID    int    `json:"source_id,omitempty" yaml:"source_id"`
	SourceType  string `json:"source_type,omitempty" yaml:"source_type" default:"article"`
	Title       string `json:"title" yaml:"title"`
	UpdatedAt   string `json:"updated_at,omitempty" yaml:"updated_at"`
	UpdatedById int    `json:"updated_by_id,omitempty" yaml:"updated_by_id"`
	URL         string `json:"url,omitempty" yaml:"url"`
}

type wrappedTranslation struct {
	Translation Translation `json:"translation"`
}

func (t *Translation) FromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	r := bytes.NewReader(b)
	b, err = frontmatter.Parse(r, &t)
	if err != nil {
		return err
	}
	t.Body = string(b)

	return nil
}

func (t *Translation) FromJson(jsonStr string) error {
	wrapped := wrappedTranslation{}
	err := json.Unmarshal([]byte(jsonStr), &wrapped)
	if err != nil {
		return err
	}
	*t = wrapped.Translation
	return nil
}

func (t *Translation) ToPayload() (string, error) {
	wrapped := wrappedTranslation{
		Translation: *t,
	}
	b, err := json.Marshal(wrapped)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *Translation) Save(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("---\n"); err != nil {
		return err
	}
	ye := yaml.NewEncoder(f)
	ye.SetIndent(2)
	if err := ye.Encode(t); err != nil {
		return err
	}
	if _, err := f.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := f.WriteString(t.Body); err != nil {
		return err
	}
	return nil
}
