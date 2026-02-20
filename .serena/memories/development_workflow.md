# zgsync Development Workflow

## Standard Development Process
1. Understand the requirement by reading CLAUDE.md first
2. Use semantic tools to explore relevant code sections
3. Make changes following existing patterns
4. Run `make test` to ensure tests pass
5. Run `make lint` to check code quality
6. Build with `make build` to verify compilation

## Common Implementation Patterns

### Adding New CLI Commands
1. Create new struct in `internal/cli/` with Kong tags
2. Embed `Global` struct for shared config
3. Implement `Run()` method
4. Register in main command struct

### Working with Zendesk API
1. Check existing methods in `internal/zendesk/`
2. Follow the pattern: JSON → Struct → Markdown (pull) or Markdown → Struct → JSON (push)
3. Use existing `FromFile()`, `FromJson()`, `Save()`, `ToPayload()` patterns

### Converter Updates
1. HTML→Markdown: Modify `internal/converter/html_to_markdown.go`
2. Markdown→HTML: Modify `internal/converter/markdown_to_html.go`
3. Always maintain compatibility with existing content