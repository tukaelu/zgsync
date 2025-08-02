---
allowed-tools: Bash, Edit, Read, TodoWrite, Grep, Glob
description: "Analyze changes and create appropriately granular commits with Conventional Commits format messages"
---

# Commit Command

## Purpose
Analyze changes and create appropriately granular commits with Conventional Commits format messages.

## Prerequisites
- Familiarity with Conventional Commits format
- Review `.claude/docs/commit-guidelines.md` for project-specific conventions

## Usage
```bash
/commit                    # Analyze all changes and create commits
/commit --dry-run         # Preview commits without creating them
/commit --interactive     # Review each commit before creating
/commit --scope=zendesk   # Add scope to commit messages
```

## Workflow

### 1. Change Analysis
- Run `git diff --cached` and `git diff` to identify all changes
- Group changes by:
  - File type/module (e.g., cli/, zendesk/, converter/)
  - Change type (feature, fix, refactor, test, docs, etc.)
  - Logical units (related functionality)

### 2. Commit Type Detection
Map changes to Conventional Commits types (see .claude/docs/commit-guidelines.md for details):
- **feat**: New features or functionality
- **fix**: Bug fixes
- **docs**: Documentation changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code changes that neither fix bugs nor add features
- **perf**: Performance improvements
- **test**: Adding or modifying tests
- **build**: Changes to build system or dependencies
- **ci**: CI/CD configuration changes
- **chore**: Other changes (gitignore, configs, etc.)

### 3. Scope Identification
Automatically detect scope from file paths:
- `internal/cli/*` → `scope: cli`
- `internal/zendesk/*` → `scope: zendesk`
- `internal/converter/*` → `scope: converter`
- `cmd/zgsync/*` → `scope: cmd`
- Multiple modules → no scope or `scope: *`

### 4. Commit Message Generation

#### Format
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

#### Examples
```
feat(zendesk): add support for article attachments

- Implement AttachmentUpload method
- Add attachment metadata to frontmatter
- Handle binary file uploads

Closes #123
```

```
fix(converter): preserve inline code blocks during HTML conversion

Goldmark was escaping backticks in inline code. Added custom
renderer to preserve original formatting.
```

```
docs: update development setup instructions

- Add Claude Code and SERENA setup
- Include troubleshooting section
- Update Go version requirement
```

### 5. Commit Granularity Rules

#### Separate Commits When:
1. **Different modules** are changed (unless tightly coupled)
2. **Different types** of changes (feat vs fix vs docs)
3. **Unrelated features** even in same module
4. **Breaking changes** (always separate with BREAKING CHANGE footer)

#### Single Commit When:
1. **Test + implementation** for same feature
2. **Related refactoring** across multiple files
3. **Single logical change** spanning multiple files

### 6. Interactive Mode Features
- Preview each proposed commit
- Edit commit messages
- Combine or split commits
- Skip commits
- Add co-authors

### 7. Validation
Before creating commits:
- Ensure commit message follows Conventional Commits spec
- Check for breaking changes indicators
- Verify file groupings make sense
- Run basic syntax checks

## Implementation

```markdown
First, read the commit guidelines:
- Check .claude/docs/commit-guidelines.md for Conventional Commits format and project-specific rules

Then analyze the git diff output and create appropriate commits following these steps:

1. Get all changes: `git status --porcelain` and `git diff`
2. Group related changes based on the rules above
3. For each group:
   - Determine commit type and scope
   - Generate descriptive message
   - Stage appropriate files
   - Create commit (with --no-verify in dry-run mode)
4. Show summary of created commits

When working with zgsync specifically, consider:
- Zendesk API changes often need tests
- Converter changes should be tested with sample Markdown/HTML
- CLI changes might need documentation updates
- Breaking changes in API client need clear documentation
```

## Advanced Options

```bash
/commit --breaking        # Mark next commit as breaking change
/commit --no-verify      # Skip pre-commit hooks
/commit --amend          # Smart amend to last commit
/commit --fixup=SHA      # Create fixup commits
```

## Examples for zgsync

### Example 1: Multiple Module Changes
**Changes:**
- Modified `internal/cli/push.go`
- Modified `internal/zendesk/article.go`
- Added `internal/zendesk/article_test.go`

**Result:**
```
Commit 1: feat(cli): add --force flag to push command
Commit 2: feat(zendesk): implement article validation
         (includes article_test.go)
```

### Example 2: Bug Fix with Tests
**Changes:**
- Fixed `internal/converter/markdown_to_html.go`
- Updated `internal/converter/markdown_to_html_test.go`
- Updated test fixtures

**Result:**
```
Commit 1: fix(converter): handle nested lists correctly

Fixed issue where nested bullet points were being flattened.
Added test cases to prevent regression.
```

### Example 3: Documentation and Code
**Changes:**
- Updated `README.md`
- Modified `internal/cli/version.go`
- Added `CHANGELOG.md`

**Result:**
```
Commit 1: feat(cli): show git commit hash in version output
Commit 2: docs: add CHANGELOG and update README

- Document version command changes
- Initialize CHANGELOG for v1.0.0
```