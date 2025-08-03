# Test Utilities (testutil)

This package provides common testing utilities and helpers to reduce code duplication across test files and improve test maintainability.

## Core Components

### 1. Error Checking (`ErrorChecker`)

Simplifies error assertion patterns in tests.

```go
// Before
if err == nil {
    t.Errorf("Expected error but got none")
} else if !strings.Contains(err.Error(), "expected text") {
    t.Errorf("Expected error containing 'expected text', got: %v", err)
}

// After
errorChecker := testutil.NewErrorChecker(t)
errorChecker.ExpectErrorContaining(err, "expected text", "operation context")
```

**Methods:**
- `ExpectError(err, context)` - Assert error should occur
- `ExpectNoError(err, context)` - Assert no error should occur  
- `ExpectErrorContaining(err, expectedText, context)` - Assert error with specific text

### 2. Assertions (`AssertionHelper`)

Provides cleaner value comparisons.

```go
// Before
if expected != actual {
    t.Errorf("Expected %v, got %v", expected, actual)
}

// After
asserter := testutil.NewAssertionHelper(t)
asserter.Equal(expected, actual, "field description")
```

**Methods:**
- `Equal(expected, actual, context)` - Assert equality
- `NotEqual(expected, actual, context)` - Assert inequality
- `Contains(haystack, needle, context)` - Assert string contains substring
- `NotEmpty(value, context)` - Assert string is not empty
- `True/False(condition, context)` - Assert boolean conditions

### 3. Test Results (`TestResult`)

Wraps operation results for cleaner error handling.

```go
// Before
result, err := someOperation()
if err != nil {
    t.Fatalf("Operation failed: %v", err)
}

// After
result := testutil.NewTestResult(someOperation()).AssertSuccess(t, "operation")
```

**Methods:**
- `AssertSuccess(t, context)` - Assert operation succeeded, return value
- `AssertError(t, context)` - Assert operation failed, return error
- `AssertErrorContaining(t, expectedText, context)` - Assert specific error

### 4. Field Comparison (`FieldComparer`)

Systematic comparison of struct fields.

```go
// Before - repetitive field comparisons
if article.Title != expected.Title {
    t.Errorf("Title: got %v, want %v", article.Title, expected.Title)
}
if article.Locale != expected.Locale {
    t.Errorf("Locale: got %v, want %v", article.Locale, expected.Locale)
}

// After - systematic comparison
fc := testutil.NewFieldComparer(t, "Article")
fc.CompareString("Title", expected.Title, article.Title)
fc.CompareString("Locale", expected.Locale, article.Locale)
fc.CompareInt("ID", expected.ID, article.ID)
fc.CompareIntPtr("UserSegmentID", expected.UserSegmentID, article.UserSegmentID)
```

**Field Types Supported:**
- `CompareString(fieldName, expected, actual)`
- `CompareInt(fieldName, expected, actual)`
- `CompareBool(fieldName, expected, actual)`
- `CompareIntPtr(fieldName, expected, actual)` - handles nil cases
- `CompareStringSlice(fieldName, expected, actual)`
- `CompareIntSlice(fieldName, expected, actual)`

### 5. File Operations (`FileHelper`)

Manages temporary files and directories for testing.

```go
// Creates temp directory, auto-cleanup
fh := testutil.NewFileHelper(t)

// Create test files
configFile := fh.CreateFile("config.yaml", "key: value")
articleFile := fh.CreateArticleTestFile("article.md", "en", 123, "Test Title")

// Assertions
fh.AssertFileExists("expected-file.md")
fh.AssertFileContains("config.yaml", "key: value")
```

**Methods:**
- `CreateFile(filename, content)` - Create file with content
- `CreateTestDataFile(filename, frontmatter, body)` - Create file with YAML frontmatter
- `CreateArticleTestFile(filename, locale, permissionGroupID, title)` - Create article test file
- `CreateTranslationTestFile(filename, locale, title, sourceID, body)` - Create translation test file
- `AssertFileExists/NotExists(filename)` - File existence assertions
- `AssertFileContent(filename, expectedContent)` - Full content comparison
- `AssertFileContains(filename, substring)` - Substring check

### 6. HTTP Testing (`HTTPHelper`)

Simplifies HTTP test server setup and validation.

```go
hh := testutil.NewHTTPHelper(t)

// Create mock responses
responses := map[string]testutil.HTTPResponse{
    "GET /api/articles/123": testutil.NewHTTPResponse(200, `{"article":{"id":123}}`),
    "POST /api/articles": testutil.NewHTTPResponse(201, `{"article":{"id":124}}`),
}
server := hh.CreateMockServer(responses)

// Zendesk-specific helpers
articleResponse := hh.CreateZendeskArticleResponse(123, "Test Article", "en", 456)
errorResponse := hh.CreateZendeskErrorResponse("ValidationError", "Title is required")
```

**Methods:**
- `CreateTestServer(handler)` - Create test server with custom handler
- `CreateMockServer(responses)` - Create server with predefined responses
- `CreateZendeskArticleResponse(id, title, locale, sectionID)` - Create article response
- `CreateZendeskTranslationResponse(id, locale, title, sourceID, body)` - Create translation response
- `AssertHTTPStatus(response, expectedStatus)` - Status assertions
- `ValidateBasicAuth(expectedAuth)` - Request validation function
- `ValidateContentType(expectedType)` - Content type validation

### 7. Advanced Comparisons

For complex data structures:

```go
// Struct comparison
sc := testutil.NewStructComparator(t)
sc.CompareStructs(expected, actual, "Article")
sc.CompareStructFields(expected, actual, []string{"Title", "Locale"}, "Article")

// Map comparison  
mc := testutil.NewMapComparator(t)
mc.CompareMaps(expectedMap, actualMap, "config")

// Error comparison
ec := testutil.NewErrorComparator(t)
ec.CompareErrors(expectedErr, actualErr, "operation")
```

## Migration Examples

### Before (Repetitive)
```go
func TestArticleFromFile(t *testing.T) {
    // ... test setup ...
    
    err := article.FromFile("test.md")
    if err != nil {
        t.Errorf("FromFile() failed: %v", err)
    }
    
    if article.Title != expected.Title {
        t.Errorf("Title: got %v, want %v", article.Title, expected.Title)
    }
    if article.Locale != expected.Locale {
        t.Errorf("Locale: got %v, want %v", article.Locale, expected.Locale)
    }
    // ... more field comparisons ...
}
```

### After (Clean)
```go
func TestArticleFromFile(t *testing.T) {
    // ... test setup ...
    
    result := testutil.NewTestResult(nil, article.FromFile("test.md"))
    result.AssertSuccess(t, "FromFile()")
    
    fc := testutil.NewFieldComparer(t, "Article")
    fc.CompareString("Title", expected.Title, article.Title)
    fc.CompareString("Locale", expected.Locale, article.Locale)
    // ... more comparisons with consistent error formatting ...
}
```

## Usage Guidelines

1. **Import once per test file**: `import "github.com/tukaelu/zgsync/internal/testutil"`

2. **Use descriptive contexts**: Always provide clear context strings for better error messages

3. **Combine helpers**: Mix and match different helpers as needed

4. **Consistent error format**: All helpers produce consistent, informative error messages

5. **Auto-cleanup**: FileHelper and HTTPHelper automatically clean up resources

## Benefits

- **🔧 Reduced duplication**: Common patterns extracted into reusable functions
- **📝 Better error messages**: Consistent, contextual error reporting  
- **🧹 Cleaner tests**: Focus on test logic rather than assertion boilerplate
- **⚡ Faster development**: Less time writing repetitive test code
- **🛡️ Fewer bugs**: Standardized patterns reduce assertion errors
- **📚 Better maintainability**: Changes to assertion logic centralized

This test utility library significantly improves the maintainability and readability of the zgsync test suite while reducing code duplication across all test files.