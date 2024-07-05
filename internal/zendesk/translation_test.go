package zendesk

import (
	"os"
	"testing"
)

func TestTranslationFromFile(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Translation
	}{
		{
			"testdata/translation-ja.md",
			Translation{
				Locale:   "ja",
				Title:    "zgsyncの使い方",
				SourceID: 12345,
				Body:     "# zgsyncの使い方\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			translation := &Translation{}
			if err := translation.FromFile(tt.filepath); err != nil {
				t.Errorf("TranslationFromFile() failed: %v", err)
			}
			if translation.Locale != tt.expected.Locale {
				t.Errorf("translation.Locale failed: got %v, want %v", translation.Locale, tt.expected.Locale)
			}
			if translation.Title != tt.expected.Title {
				t.Errorf("translation.Title failed: got %v, want %v", translation.Title, tt.expected.Title)
			}
			if translation.SourceID != tt.expected.SourceID {
				t.Errorf("translation.SourceId failed: got %v, want %v", translation.SourceID, tt.expected.SourceID)
			}
			if translation.Body != tt.expected.Body {
				t.Errorf("translation.Body failed: got %v, want %v", translation.Body, tt.expected.Body)
			}
		})
	}
}

func TestTranslationFromJson(t *testing.T) {
	var tests = []struct {
		filepath string
		expected Translation
	}{
		{
			"testdata/translation.json",
			Translation{
				Body:   "# zgsyncの使い方\n",
				Locale: "ja",
				Title:  "zgsyncの使い方",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			translation := &Translation{}
			jsonContent, _ := os.ReadFile(tt.filepath)
			if err := translation.FromJson(string(jsonContent)); err != nil {
				t.Errorf("TranslationFromJson() failed: %v", err)
			}
			if translation.Locale != tt.expected.Locale {
				t.Errorf("translation.Locale failed: got %v, want %v", translation.Locale, tt.expected.Locale)
			}
			if translation.Title != tt.expected.Title {
				t.Errorf("translation.Title failed: got %v, want %v", translation.Title, tt.expected.Title)
			}
			if translation.Body != tt.expected.Body {
				t.Errorf("translation.Body failed: got %v, want %v", translation.Body, tt.expected.Body)
			}
		})
	}
}
