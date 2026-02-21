package zendesk

import (
	"fmt"
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

func (ds *MockDataStore) deleteArticle(id int) bool {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if _, exists := ds.articles[id]; !exists {
		return false
	}
	delete(ds.articles, id)
	return true
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
