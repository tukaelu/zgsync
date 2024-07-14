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

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/
type Article struct {
	AuthorID          int      `json:"author_id,omitempty" yaml:"author_id"`
	Body              string   `json:"body,omitempty" yaml:"-"`
	CommentsDisabled  bool     `json:"comments_disabled,omitempty" yaml:"comments_disabled"`
	ContentTagIDs     []string `json:"content_tag_ids,omitempty" yaml:"content_tag_ids"`
	CreatedAt         string   `json:"created_at,omitempty" yaml:"created_at"`
	Draft             bool     `json:"draft,omitempty" yaml:"draft"`
	EditedAt          string   `json:"edited_at,omitempty" yaml:"edited_at"`
	HtmlURL           string   `json:"html_url,omitempty" yaml:"html_url"`
	ID                int      `json:"id,omitempty" yaml:"id"`
	LabelNames        []string `json:"label_names,omitempty" yaml:"label_names"`
	Locale            string   `json:"locale" yaml:"locale"`
	Outdated          bool     `json:"outdated,omitempty" yaml:"outdated"`
	OutdatedLocales   []string `json:"outdated_locales,omitempty" yaml:"outdated_locales"`
	PermissionGroupID int      `json:"permission_group_id,omitempty" yaml:"permission_group_id"`
	Position          int      `json:"position,omitempty" yaml:"position"`
	Promoted          bool     `json:"promoted,omitempty" yaml:"promoted"`
	SectionID         int      `json:"section_id,omitempty" yaml:"section_id"`
	SourceLocale      string   `json:"source_locale,omitempty" yaml:"source_locale"`
	Title             string   `json:"title" yaml:"title"`
	UpdatedAt         string   `json:"updated_at,omitempty" yaml:"updated_at"`
	Url               string   `json:"url,omitempty" yaml:"url"`
	UserSegmentID     int      `json:"user_segment_id" yaml:"user_segment_id"`
	UserSegmentIDs    []int    `json:"user_segment_ids,omitempty" yaml:"user_segment_ids"`
	VoteCount         int      `json:"vote_count,omitempty" yaml:"vote_count"`
	VoteSum           int      `json:"vote_sum,omitempty" yaml:"vote_sum"`
}

type wrappedArticle struct {
	Article           Article `json:"article"`
	NotifySubscribers bool    `json:"notify_subscribers,omitempty" default:"false"`
}

func (a *Article) FromFile(path string) error {
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
	_, err = frontmatter.Parse(r, &a)
	if err != nil {
		return err
	}
	return nil
}

func (a *Article) FromJson(jsonStr string) error {
	wrapped := wrappedArticle{}
	err := json.Unmarshal([]byte(jsonStr), &wrapped)
	if err != nil {
		return err
	}
	*a = wrapped.Article
	return nil
}

func (a *Article) Save(path string, appendFileName bool) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	if appendFileName {
		path = filepath.Join(path, strconv.Itoa(a.ID)+".md")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("---\n"); err != nil {
		return err
	}
	ye := yaml.NewEncoder(f)
	ye.SetIndent(2)
	if err := ye.Encode(a); err != nil {
		return err
	}
	if _, err := f.WriteString("---\n"); err != nil {
		return err
	}
	return nil
}

func (a *Article) ToPayload(notify bool) (string, error) {
	wrapped := wrappedArticle{
		Article:           *a,
		NotifySubscribers: notify,
	}
	b, err := json.Marshal(wrapped)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
