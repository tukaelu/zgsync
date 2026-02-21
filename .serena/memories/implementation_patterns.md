# Implementation Patterns for zgsync

## Testing Patterns

### Table-Driven Tests
```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    // test cases
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test implementation
    })
}
```

### Error Handling Pattern
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

## Zendesk API Patterns

### File Naming Convention
- Translations: `{ArticleID}-{Locale}.md` (e.g., `123456-ja.md`)
- Articles: `{ArticleID}.md` (e.g., `123456.md`)

### Frontmatter Structure
```yaml
---
title: Article Title
locale: ja
draft: true
section_id: 123456
---
```

### API Response Handling
1. Parse JSON response into struct
2. Convert HTML body to Markdown using converter
3. Save with appropriate frontmatter

## CLI Command Pattern
```go
type CommandName struct {
    Global
    // command-specific fields with kong tags
}

func (c *CommandName) Run() error {
    // implementation
}
```

## Converter Patterns
- Use goldmark for Markdown→HTML (CommonMark compliant)
- Support Pandoc-style divs: `:::{.class}`
- Support heading attributes: `## Title {#id}`
- Preserve link attributes during conversion