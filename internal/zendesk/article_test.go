package zendesk

import (
	"os"
	"testing"
)

func TestArticleFromFile(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Article
	}{
		{
			"testdata/article-ja.md",
			Article{
				Locale:            "ja",
				PermissionGroupID: 12345,
				Title:             "zgsyncの使い方",
				UserSegmentIDs:    []int{123, 456},
			},
		},
		{
			"testdata/article-en.md",
			Article{
				Locale:            "en_us",
				PermissionGroupID: 56,
				Title:             "How to use zgsync",
				UserSegmentID:     123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			article := &Article{}
			if err := article.FromFile(tt.filepath); err != nil {
				t.Errorf("ArticleFromFile() failed: %v", err)
			}
			if article.Locale != tt.expected.Locale {
				t.Errorf("article.Locale failed: got %v, want %v", article.Locale, tt.expected.Locale)
			}
			if article.PermissionGroupID != tt.expected.PermissionGroupID {
				t.Errorf("article.PermissionGroupId failed: got %v, want %v", article.PermissionGroupID, tt.expected.PermissionGroupID)
			}
			if article.Title != tt.expected.Title {
				t.Errorf("article.Title failed: got %v, want %v", article.Title, tt.expected.Title)
			}
			if article.UserSegmentID != tt.expected.UserSegmentID {
				t.Errorf("article.UserSegmentId failed: got %v, want %v", article.UserSegmentID, tt.expected.UserSegmentID)
			}
			if len(article.UserSegmentIDs) != len(tt.expected.UserSegmentIDs) {
				t.Errorf("article.UserSegmentIds failed: got %v, want %v", article.UserSegmentIDs, tt.expected.UserSegmentIDs)
			}
		})
	}
}

func TestArticleFromJson(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Article
	}{
		{
			"testdata/article.json",
			Article{
				AuthorID:          3465,
				CommentsDisabled:  true,
				ContentTagIDs:     []string{"01GT23D51Y", "01GT23FWWN"},
				ID:                37486578,
				Locale:            "en_us",
				PermissionGroupID: 123,
				Position:          42,
				Promoted:          false,
				Title:             "How to use zgsync",
				UserSegmentID:     12,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			article := &Article{}
			jsonContent, _ := os.ReadFile(tt.filepath)
			if err := article.FromJson(string(jsonContent)); err != nil {
				t.Errorf("ArticleFromJson() failed: %v", err)
			}
			if article.AuthorID != tt.expected.AuthorID {
				t.Errorf("article.AuthorID failed: got %v, want %v", article.AuthorID, tt.expected.AuthorID)
			}
			if article.CommentsDisabled != tt.expected.CommentsDisabled {
				t.Errorf("article.CommentsDisabled failed: got %v, want %v", article.CommentsDisabled, tt.expected.CommentsDisabled)
			}
			if len(article.ContentTagIDs) != len(tt.expected.ContentTagIDs) {
				t.Errorf("article.ContentTagIds failed: got %v, want %v", article.ContentTagIDs, tt.expected.ContentTagIDs)
			}
			if article.ID != tt.expected.ID {
				t.Errorf("article.ID failed: got %v, want %v", article.ID, tt.expected.ID)
			}
			if article.Locale != tt.expected.Locale {
				t.Errorf("article.Locale failed: got %v, want %v", article.Locale, tt.expected.Locale)
			}
			if article.PermissionGroupID != tt.expected.PermissionGroupID {
				t.Errorf("article.PermissionGroupId failed: got %v, want %v", article.PermissionGroupID, tt.expected.PermissionGroupID)
			}
			if article.Position != tt.expected.Position {
				t.Errorf("article.Position failed: got %v, want %v", article.Position, tt.expected.Position)
			}
			if article.Promoted != tt.expected.Promoted {
				t.Errorf("article.Promoted failed: got %v, want %v", article.Promoted, tt.expected.Promoted)
			}
			if article.Title != tt.expected.Title {
				t.Errorf("article.Title failed: got %v, want %v", article.Title, tt.expected.Title)
			}
			if article.UserSegmentID != tt.expected.UserSegmentID {
				t.Errorf("article.UserSegmentId failed: got %v, want %v", article.UserSegmentID, tt.expected.UserSegmentID)
			}
		})
	}
}
