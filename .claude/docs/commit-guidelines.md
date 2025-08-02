# Commit Guidelines for zgsync

## Quick Reference

### Conventional Commits Format
```
<type>(<scope>): <subject>

[body]

[footer]
```

### Types
- `feat`: New feature
- `fix`: Bug fix  
- `docs`: Documentation only
- `style`: Code style (formatting, whitespace)
- `refactor`: Code restructuring without changing behavior
- `perf`: Performance improvement
- `test`: Adding/updating tests
- `build`: Build system or dependencies
- `ci`: CI/CD configuration
- `chore`: Maintenance tasks

### Scopes for zgsync
- `cli`: CLI commands and interface
- `zendesk`: Zendesk API client
- `converter`: Markdown/HTML conversion
- `cmd`: Main application entry
- `config`: Configuration handling
- `*`: Multiple scopes affected

## Commit Command Workflow

### Basic Usage
```bash
# Analyze and create commits automatically
/commit

# Preview without committing
/commit --dry-run

# Interactive mode (review each commit)
/commit --interactive
```

### Examples

#### Feature Addition
```bash
# After implementing article draft status
/commit
# Creates: "feat(zendesk): add draft status support for articles"
```

#### Bug Fix
```bash
# After fixing HTML conversion issue
/commit
# Creates: "fix(converter): preserve link attributes during conversion"
```

#### Multiple Changes
```bash
# After updating docs and fixing a bug
/commit
# Creates two commits:
# 1. "fix(cli): handle empty article IDs gracefully"
# 2. "docs: update troubleshooting guide"
```

## Commit Message Best Practices

### Subject Line
- Use imperative mood ("add", not "added" or "adds")
- Don't capitalize first letter
- No period at the end
- Limit to 50 characters

### Body (when needed)
- Explain **what** and **why**, not how
- Wrap at 72 characters
- Separate from subject with blank line

### Examples of Good Commits

```
feat(zendesk): add batch article update support

Implement BatchUpdate method to update multiple articles in a single
API call. This reduces API rate limit consumption when syncing large
numbers of articles.

Closes #45
```

```
fix(converter): handle special characters in code blocks

Goldmark was incorrectly escaping angle brackets inside code blocks,
breaking HTML/XML examples. Added custom renderer to preserve raw
content within code fences.
```

```
refactor(cli): extract common validation logic

Move shared validation functions to internal/cli/validate.go to
reduce code duplication across push, pull, and empty commands.
```

## Breaking Changes

When making breaking changes:

```
feat(zendesk)!: change Article.ID type from string to int64

BREAKING CHANGE: Article.ID is now int64 instead of string.
This aligns with Zendesk API documentation and prevents type
conversion errors.

Migration: Update any code that expects Article.ID to be string.
```

## Commit Grouping Guidelines

### Group Together
- Implementation + its tests
- Related refactoring in same module
- Documentation for a specific feature

### Separate Commits
- Different modules (cli vs zendesk)
- Different types (feat vs fix)
- Unrelated changes even in same file
- Breaking changes from other changes

## Special Cases

### Work in Progress
```bash
# Save WIP without committing
git stash save "WIP: implementing attachment support"

# Or create WIP commit (to be amended later)
git commit -m "WIP: attachment upload logic"
```

### Fixing Previous Commit
```bash
# Create fixup commit
/commit --fixup=abc123

# Or amend if it's the last commit
/commit --amend
```

### Co-authored Commits
```bash
/commit --co-author="Name <email>"
```

## Tips

1. **Run tests before committing**: `make test`
2. **Check linting**: `make lint`
3. **Use dry-run first**: `/commit --dry-run`
4. **Keep commits focused**: One logical change per commit
5. **Write for future readers**: Clear messages help debugging