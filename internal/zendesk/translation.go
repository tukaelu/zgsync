package zendesk

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/#update-translation
type Translation struct {
	Title       string `json:"title" yaml:"title"`
	Locale      string `json:"locale" yaml:"locale"`
	Draft       bool   `json:"draft,omitempty" yaml:"draft"`
	Outdated    bool   `json:"outdated,omitempty" yaml:"outdated"`
	SectionID   int    `json:"-" yaml:"section_id,omitempty"`
	SourceID    int    `json:"source_id,omitempty" yaml:"source_id"`
	HtmlURL     string `json:"html_url,omitempty" yaml:"html_url"`
	CreatedAt   string `json:"created_at,omitempty" yaml:"-"`
	UpdatedAt   string `json:"updated_at,omitempty" yaml:"-"`
	ID          int    `json:"id" yaml:"-"`
	URL         string `json:"url,omitempty" yaml:"-"`
	SourceType  string `json:"source_type,omitempty" yaml:"-"`
	CreatedById int    `json:"created_by_id,omitempty" yaml:"-"`
	UpdatedById int    `json:"updated_by_id,omitempty" yaml:"-"`
	Body        string `json:"body,omitempty" yaml:"-"`
}

type wrappedTranslation struct {
	Translation Translation `json:"translation"`
}

func (t *Translation) FromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

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

func (t *Translation) Save(path string, appendFileName bool) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	if appendFileName {
		path = filepath.Join(path, strconv.Itoa(t.SourceID)+"-"+t.Locale+".md")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

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
