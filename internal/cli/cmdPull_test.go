package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	"github.com/tukaelu/zgsync/internal/converter"
)

func TestCommandPull_Run(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		cmd           CommandPull
		expectError   bool
		mockSetup     func(*testhelper.MockZendeskClient)
		validateFiles func(*testing.T, string)
	}{
		{
			name: "pull translation successfully",
			cmd: CommandPull{
				Locale:         testhelper.TestLocales.Japanese,
				Raw:            false,
				SaveArticle:    false,
				WithSectionDir: false,
				ArticleIDs:     []int{testhelper.TestArticleID},
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				expectedFile := filepath.Join(dir, fmt.Sprintf("%d-%s.md", testhelper.TestArticleID, testhelper.TestLocales.Japanese))
				if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", expectedFile)
				}
			},
		},
		{
			name: "pull with save article",
			cmd: CommandPull{
				Locale:         testhelper.TestLocales.English,
				Raw:            false,
				SaveArticle:    true,
				WithSectionDir: false,
				ArticleIDs:     []int{testhelper.TestArticleID},
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					article := testhelper.Article{
						ID:        articleID,
						SectionID: testhelper.TestSectionID,
						Title:     "Test Article",
						Locale:    locale,
					}
					response := map[string]testhelper.Article{"article": article}
					jsonBytes, _ := json.Marshal(response)
					return string(jsonBytes), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				translationFile := filepath.Join(dir, fmt.Sprintf("%d-%s.md", testhelper.TestArticleID, testhelper.TestLocales.English))
				if _, err := os.Stat(translationFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", translationFile)
				}
				articleFile := filepath.Join(dir, fmt.Sprintf("%d.md", testhelper.TestArticleID))
				if _, err := os.Stat(articleFile); os.IsNotExist(err) {
					t.Errorf("Expected article file %s to exist", articleFile)
				}
			},
		},
		{
			name: "pull with section directory",
			cmd: CommandPull{
				Locale:         testhelper.TestLocales.Japanese,
				Raw:            false,
				SaveArticle:    false,
				WithSectionDir: true,
				ArticleIDs:     []int{testhelper.TestArticleID},
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				expectedFile := filepath.Join(dir, fmt.Sprintf("%d", testhelper.TestSectionID), fmt.Sprintf("%d-%s.md", testhelper.TestArticleID, testhelper.TestLocales.Japanese))
				if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", expectedFile)
				}
			},
		},
		{
			name: "pull raw mode",
			cmd: CommandPull{
				Locale:         testhelper.TestLocales.Japanese,
				Raw:            true,
				SaveArticle:    false,
				WithSectionDir: false,
				ArticleIDs:     []int{testhelper.TestArticleID},
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					translation := testhelper.Translation{
						ID:       1,
						Title:    "Test Translation",
						Body:     "<h1>Raw HTML</h1>",
						Locale:   locale,
						SourceID: articleID,
					}
					response := map[string]testhelper.Translation{"translation": translation}
					jsonBytes, _ := json.Marshal(response)
					return string(jsonBytes), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				expectedFile := filepath.Join(dir, fmt.Sprintf("%d-%s.md", testhelper.TestArticleID, testhelper.TestLocales.Japanese))
				if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", expectedFile)
				}
				
				content, err := os.ReadFile(expectedFile)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
					return
				}
				
				if !strings.Contains(string(content), "<h1>Raw HTML</h1>") {
					t.Errorf("Expected raw HTML content to be preserved")
				}
			},
		},
		{
			name: "multiple article IDs",
			cmd: CommandPull{
				Locale:         testhelper.TestLocales.Japanese,
				Raw:            false,
				SaveArticle:    false,
				WithSectionDir: false,
				ArticleIDs:     []int{123, 456},
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.ShowArticleFunc = func(locale string, articleID int) (string, error) {
					return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				for _, id := range []int{123, 456} {
					expectedFile := filepath.Join(dir, fmt.Sprintf("%d-%s.md", id, testhelper.TestLocales.Japanese))
					if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
						t.Errorf("Expected translation file for ID %d to exist", id)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tempDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)
			
			cmd := tt.cmd
			cmd.client = mockClient
			cmd.converter = converter.NewConverter(false)

			global := &Global{
				Config: Config{
					DefaultLocale: testhelper.TestLocales.English,
					ContentsDir:   testDir,
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if !tt.expectError {
				tt.validateFiles(t, testDir)
			}
		})
	}
}

func TestCommandPull_AfterApply(t *testing.T) {
	global := &Global{
		Config: Config{
			Subdomain:             "test",
			Email:                 "test@example.com",
			Token:                 "token",
			EnableLinkTargetBlank: true,
		},
	}

	cmd := &CommandPull{}
	err := cmd.AfterApply(global)
	
	if err != nil {
		t.Errorf("AfterApply() failed: %v", err)
	}
	
	if cmd.client == nil {
		t.Error("client should be initialized")
	}
	
	if cmd.converter == nil {
		t.Error("converter should be initialized")
	}
}

func TestCommandPull_DefaultLocale(t *testing.T) {
	tempDir := t.TempDir()
	
	mockClient := &testhelper.MockZendeskClient{}
	mockClient.ShowArticleFunc = func(locale string, articleID int) (string, error) {
		if locale != testhelper.TestLocales.French {
			t.Errorf("Expected locale '%s', got '%s'", testhelper.TestLocales.French, locale)
		}
		return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
	}
	mockClient.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
		if locale != testhelper.TestLocales.French {
			t.Errorf("Expected locale '%s', got '%s'", testhelper.TestLocales.French, locale)
		}
		return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
	}
	
	cmd := &CommandPull{
		Locale:         "", // Empty locale should use default
		ArticleIDs:     []int{testhelper.TestArticleID},
		client:         mockClient,
		converter:      converter.NewConverter(false),
	}

	global := &Global{
		Config: Config{
			DefaultLocale: testhelper.TestLocales.French, // Default locale
			ContentsDir:   tempDir,
		},
	}

	err := cmd.Run(global)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

func TestCommandPull_FileSystemErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func() string
		expectError bool
	}{
		{
			name: "non-existent contents directory",
			setupDir: func() string {
				return "/non/existent/directory"
			},
			expectError: true,
		},
		{
			name: "read-only contents directory",
			setupDir: func() string {
				tempDir := t.TempDir()
				// Make directory read-only (no write permissions)
				if err := os.Chmod(tempDir, 0444); err != nil {
					t.Skipf("Cannot change directory permissions on this platform: %v", err)
				}
				return tempDir
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentsDir := tt.setupDir()
			
			// Restore permissions for cleanup if needed
			defer func() {
				if tt.name == "read-only contents directory" {
					_ = os.Chmod(contentsDir, 0755)
				}
			}()

			mockClient := &testhelper.MockZendeskClient{}
			mockClient.ShowArticleFunc = func(locale string, articleID int) (string, error) {
				return testhelper.CreateDefaultArticleResponse(articleID, testhelper.TestSectionID), nil
			}
			mockClient.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
				return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
			}

			cmd := &CommandPull{
				Locale:      testhelper.TestLocales.Japanese,
				ArticleIDs:  []int{testhelper.TestArticleID},
				client:      mockClient,
				converter:   converter.NewConverter(false),
			}

			global := &Global{
				Config: Config{
					DefaultLocale: testhelper.TestLocales.English,
					ContentsDir:   contentsDir,
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}