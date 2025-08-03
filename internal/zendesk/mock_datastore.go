package zendesk

import (
	"fmt"
	"time"
)

// initializeDefaultData populates the data store with initial test data
func (ds *MockDataStore) initializeDefaultData() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	ds.initializeDefaultDataUnsafe()
}

// initializeDefaultDataUnsafe populates the data store without taking locks
func (ds *MockDataStore) initializeDefaultDataUnsafe() {

	// Initialize counters
	ds.nextID.article = 1000
	ds.nextID.translation = 2000
	ds.nextID.section = 100
	ds.nextID.user = 500

	// Create default sections
	ds.sections[123] = &MockSection{
		ID:          123,
		Name:        "Getting Started",
		Description: "Basic information for new users",
		CategoryID:  1,
		Locale:      "en_us",
		Position:    1,
	}

	ds.sections[456] = &MockSection{
		ID:          456,
		Name:        "Advanced Topics",
		Description: "In-depth guides for experienced users",
		CategoryID:  1,
		Locale:      "en_us",
		Position:    2,
	}

	ds.sections[789] = &MockSection{
		ID:          789,
		Name:        "API Documentation",
		Description: "Technical documentation for developers",
		CategoryID:  2,
		Locale:      "en_us",
		Position:    1,
	}

	// Create default users
	ds.users[1] = &MockUser{
		ID:    1,
		Name:  "Test Agent",
		Email: "agent@example.com",
		Role:  "agent",
	}

	ds.users[2] = &MockUser{
		ID:    2,
		Name:  "Test Admin",
		Email: "admin@example.com", 
		Role:  "admin",
	}

	// Create some default articles
	defaultArticle := &Article{
		ID:                456,
		Title:             "Sample Article",
		Locale:            "en_us",
		SectionID:         123,
		PermissionGroupID: 1,
		AuthorID:          1,
		Draft:             false,
		Position:          1,
		Promoted:          false,
		CommentsDisabled:  false,
		ContentTagIDs:     []string{"tag1", "tag2"},
	}
	ds.articles[456] = defaultArticle

	// Create default translation for the article
	defaultTranslation := &Translation{
		ID:       2000,
		SourceID: 456,
		Locale:   "ja",
		Title:    "サンプル記事",
		Body:     "<h1>これはサンプルです</h1><p>テスト内容</p>",
		Draft:    false,
	}
	ds.translations["456-ja"] = defaultTranslation
}

// Article operations

func (ds *MockDataStore) articleExists(id int) bool {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	_, exists := ds.articles[id]
	return exists
}

func (ds *MockDataStore) getArticle(id int) *Article {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	
	article, exists := ds.articles[id]
	if !exists {
		return nil
	}
	
	// Return a copy to prevent external modification
	articleCopy := *article
	return &articleCopy
}

func (ds *MockDataStore) createArticle(locale string, sectionID int) *Article {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	id := ds.nextID.article
	ds.nextID.article++

	article := &Article{
		ID:                id,
		Title:             fmt.Sprintf("Article %d", id),
		Locale:            locale,
		SectionID:         sectionID,
		PermissionGroupID: 1,
		AuthorID:          1,
		Draft:             true,
		Position:          1,
		Promoted:          false,
		CommentsDisabled:  false,
		ContentTagIDs:     []string{},
	}

	ds.articles[id] = article
	
	// Return a copy
	articleCopy := *article
	return &articleCopy
}

func (ds *MockDataStore) updateArticle(id int) *Article {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	article, exists := ds.articles[id]
	if !exists {
		return nil
	}

	// Simulate update by modifying title
	article.Title = fmt.Sprintf("Updated Article %d", id)
	
	// Return a copy
	articleCopy := *article
	return &articleCopy
}





// Translation operations



func (ds *MockDataStore) getTranslation(articleID int, locale string) *Translation {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	key := fmt.Sprintf("%d-%s", articleID, locale)
	translation, exists := ds.translations[key]
	if !exists {
		return nil
	}
	
	// Return a copy to prevent external modification
	translationCopy := *translation
	return &translationCopy
}

func (ds *MockDataStore) createTranslation(articleID int, locale string) *Translation {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Verify article exists
	if _, exists := ds.articles[articleID]; !exists {
		return nil
	}

	id := ds.nextID.translation
	ds.nextID.translation++

	translation := &Translation{
		ID:       id,
		SourceID: articleID,
		Locale:   locale,
		Title:    fmt.Sprintf("Translation %d (%s)", id, locale),
		Body:     fmt.Sprintf("<h1>Translation Content %d</h1><p>Content in %s</p>", id, locale),
		Draft:    true,
	}

	key := fmt.Sprintf("%d-%s", articleID, locale)
	ds.translations[key] = translation
	
	// Return a copy
	translationCopy := *translation
	return &translationCopy
}

func (ds *MockDataStore) updateTranslation(articleID int, locale string) *Translation {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	key := fmt.Sprintf("%d-%s", articleID, locale)
	translation, exists := ds.translations[key]
	if !exists {
		return nil
	}

	// Simulate update by modifying title
	translation.Title = fmt.Sprintf("Updated Translation %d (%s)", translation.ID, locale)
	
	// Return a copy
	translationCopy := *translation
	return &translationCopy
}





// Section operations

func (ds *MockDataStore) sectionExists(id int) bool {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	_, exists := ds.sections[id]
	return exists
}





// User operations





// Utility methods

// Reset clears all data and reinitializes with defaults
func (ds *MockDataStore) Reset() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Clear all maps
	ds.articles = make(map[int]*Article)
	ds.translations = make(map[string]*Translation)
	ds.sections = make(map[int]*MockSection)
	ds.users = make(map[int]*MockUser)

	// Reinitialize with defaults (without taking lock again)
	ds.initializeDefaultDataUnsafe()
}

// GetStats returns statistics about the current data store
func (ds *MockDataStore) GetStats() map[string]int {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	return map[string]int{
		"articles":     len(ds.articles),
		"translations": len(ds.translations),
		"sections":     len(ds.sections),
		"users":        len(ds.users),
	}
}

// ValidateRelationships checks data integrity
func (ds *MockDataStore) ValidateRelationships() []string {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	var issues []string

	// Check that all articles reference valid sections
	for _, article := range ds.articles {
		if _, exists := ds.sections[article.SectionID]; !exists {
			issues = append(issues, fmt.Sprintf("Article %d references non-existent section %d", article.ID, article.SectionID))
		}

		if _, exists := ds.users[article.AuthorID]; !exists {
			issues = append(issues, fmt.Sprintf("Article %d references non-existent author %d", article.ID, article.AuthorID))
		}
	}

	// Check that all translations reference valid articles
	for key, translation := range ds.translations {
		if _, exists := ds.articles[translation.SourceID]; !exists {
			issues = append(issues, fmt.Sprintf("Translation %s references non-existent article %d", key, translation.SourceID))
		}
	}

	return issues
}

// Backup creates a snapshot of the current data
func (ds *MockDataStore) Backup() *MockDataStoreSnapshot {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	snapshot := &MockDataStoreSnapshot{
		Timestamp:    time.Now(),
		Articles:     make(map[int]*Article),
		Translations: make(map[string]*Translation),
		Sections:     make(map[int]*MockSection),
		Users:        make(map[int]*MockUser),
	}

	// Deep copy all data
	for id, article := range ds.articles {
		articleCopy := *article
		snapshot.Articles[id] = &articleCopy
	}

	for key, translation := range ds.translations {
		translationCopy := *translation
		snapshot.Translations[key] = &translationCopy
	}

	for id, section := range ds.sections {
		sectionCopy := *section
		snapshot.Sections[id] = &sectionCopy
	}

	for id, user := range ds.users {
		userCopy := *user
		snapshot.Users[id] = &userCopy
	}

	return snapshot
}

// MockDataStoreSnapshot represents a point-in-time snapshot of the data store
type MockDataStoreSnapshot struct {
	Timestamp    time.Time
	Articles     map[int]*Article
	Translations map[string]*Translation
	Sections     map[int]*MockSection
	Users        map[int]*MockUser
}

// Restore loads data from a snapshot
func (ds *MockDataStore) Restore(snapshot *MockDataStoreSnapshot) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Clear current data
	ds.articles = make(map[int]*Article)
	ds.translations = make(map[string]*Translation)
	ds.sections = make(map[int]*MockSection)
	ds.users = make(map[int]*MockUser)

	// Load from snapshot
	for id, article := range snapshot.Articles {
		articleCopy := *article
		ds.articles[id] = &articleCopy
	}

	for key, translation := range snapshot.Translations {
		translationCopy := *translation
		ds.translations[key] = &translationCopy
	}

	for id, section := range snapshot.Sections {
		sectionCopy := *section
		ds.sections[id] = &sectionCopy
	}

	for id, user := range snapshot.Users {
		userCopy := *user
		ds.users[id] = &userCopy
	}
}