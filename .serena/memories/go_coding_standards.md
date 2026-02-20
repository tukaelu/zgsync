# Go Coding Standards for zgsync

## Code Organization
- **Error Handling**: Always return errors, use `fmt.Errorf` with `%w` for wrapping
- **Struct Embedding**: Use for composition (e.g., CLI commands embed `Global`)
- **Method Receivers**: Use pointer receivers for methods that modify state
- **Testing**: Table-driven tests with descriptive test case names

## Naming Conventions
- **Exported**: PascalCase (e.g., `Article`, `FromFile`)
- **Private**: camelCase (e.g., `parseConfig`)
- **Test Helpers**: Start with `test` prefix (e.g., `testLoadFile`)

## Import Style
```go
import (
    // Standard library
    "fmt"
    "os"
    
    // Third-party packages
    "github.com/adrg/frontmatter"
    
    // Internal packages
    "github.com/tukaelu/zgsync/internal/zendesk"
)
```

## Common Idioms
- Defer cleanup immediately after resource acquisition
- Check errors immediately after function calls
- Use early returns to reduce nesting
- Prefer explicit over implicit