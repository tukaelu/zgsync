package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
)

func TestCommandEmpty_Run(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		cmd           CommandEmpty
		expectError   bool
		mockSetup     func(*testhelper.MockZendeskClient)
		validateFiles func(*testing.T, string)
	}{
		{
			name: "create empty article successfully",
			cmd: CommandEmpty{
				SectionID:         testhelper.TestSectionID,
				Title:             "New Article",
				Locale:            testhelper.TestLocales.Japanese,
				PermissionGroupID: testhelper.TestPermissionGroupID,
				UserSegmentID:     nil,
				SaveArticle:       true,
				WithSectionDir:    false,
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					if sectionID != testhelper.TestSectionID {
						t.Errorf("Expected section ID %d, got %d", testhelper.TestSectionID, sectionID)
					}
					if locale != testhelper.TestLocales.Japanese {
						t.Errorf("Expected locale '%s', got '%s'", testhelper.TestLocales.Japanese, locale)
					}
					return testhelper.CreateDefaultArticleResponse(789, sectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				translationFile := filepath.Join(dir, "789-ja.md")
				if _, err := os.Stat(translationFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", translationFile)
				}
				articleFile := filepath.Join(dir, "789.md")
				if _, err := os.Stat(articleFile); os.IsNotExist(err) {
					t.Errorf("Expected article file %s to exist", articleFile)
				}
			},
		},
		{
			name: "create empty article with section directory",
			cmd: CommandEmpty{
				SectionID:         testhelper.TestSectionID,
				Title:             "New Article",
				Locale:            testhelper.TestLocales.English,
				PermissionGroupID: testhelper.TestPermissionGroupID,
				UserSegmentID:     nil,
				SaveArticle:       false,
				WithSectionDir:    true,
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					return testhelper.CreateDefaultArticleResponse(111, sectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				translationFile := filepath.Join(dir, fmt.Sprintf("%d", testhelper.TestSectionID), "111-en_us.md")
				if _, err := os.Stat(translationFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", translationFile)
				}
				// Article file should not exist when SaveArticle is false
				articleFile := filepath.Join(dir, fmt.Sprintf("%d", testhelper.TestSectionID), "111.md")
				if _, err := os.Stat(articleFile); !os.IsNotExist(err) {
					t.Errorf("Article file %s should not exist when SaveArticle is false", articleFile)
				}
			},
		},
		{
			name: "create empty article with default values",
			cmd: CommandEmpty{
				SectionID:         testhelper.TestSectionID,
				Title:             "Default Article",
				Locale:            "", // Should use default
				PermissionGroupID: 0,  // Should use default
				UserSegmentID:     nil,
				SaveArticle:       false,
				WithSectionDir:    false,
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					if locale != testhelper.TestLocales.French {
						t.Errorf("Expected default locale '%s', got '%s'", testhelper.TestLocales.French, locale)
					}
					return testhelper.CreateDefaultArticleResponse(222, sectionID), nil
				}
				mock.ShowTranslationFunc = func(articleID int, locale string) (string, error) {
					return testhelper.CreateDefaultTranslationResponse(1, articleID, locale), nil
				}
			},
			validateFiles: func(t *testing.T, dir string) {
				translationFile := filepath.Join(dir, "222-fr.md")
				if _, err := os.Stat(translationFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", translationFile)
				}
			},
		},
		{
			name: "create article with user segment ID",
			cmd: CommandEmpty{
				SectionID:         testhelper.TestSectionID,
				Title:             "Segment Article",
				Locale:            testhelper.TestLocales.Japanese,
				PermissionGroupID: testhelper.TestPermissionGroupID,
				UserSegmentID:     testhelper.IntPtr(testhelper.TestUserSegmentID),
				SaveArticle:       false,
				WithSectionDir:    false,
			},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				mock.CreateArticleFunc = func(locale string, sectionID int, payload string) (string, error) {
					// Verify payload contains user_segment_id using proper JSON parsing
					var payloadData map[string]interface{}
					if err := json.Unmarshal([]byte(payload), &payloadData); err != nil {
						t.Errorf("Failed to parse payload JSON: %v", err)
						return "", err
					}

					if article, ok := payloadData["article"].(map[string]interface{}); ok {
						if userSegmentID, exists := article["user_segment_id"]; !exists {
							t.Error("Expected payload to contain user_segment_id")
						} else if int(userSegmentID.(float64)) != testhelper.TestUserSegmentID {
							t.Errorf("Expected user_segment_id %d, got %v", testhelper.TestUserSegmentID, userSegmentID)
						}
					} else {
						t.Error("Payload should contain article object")
					}

					article := testhelper.Article{
						ID:        333,
						SectionID: sectionID,
						Title:     "Segment Article",
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
				translationFile := filepath.Join(dir, "333-ja.md")
				if _, err := os.Stat(translationFile); os.IsNotExist(err) {
					t.Errorf("Expected translation file %s to exist", translationFile)
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

			global := &Global{
				Config: Config{
					DefaultLocale:            testhelper.TestLocales.French,
					DefaultPermissionGroupID: testhelper.TestPermissionGroupID,
					DefaultCommentsDisabled:  true,
					NotifySubscribers:        false,
					ContentsDir:              testDir,
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

func TestCommandEmpty_AfterApply(t *testing.T) {
	global := &Global{
		Config: Config{
			Subdomain: "test",
			Email:     "test@example.com",
			Token:     "token",
		},
	}

	cmd := &CommandEmpty{}
	err := cmd.AfterApply(global)

	if err != nil {
		t.Errorf("AfterApply() failed: %v", err)
	}

	if cmd.client == nil {
		t.Error("client should be initialized")
	}
}
