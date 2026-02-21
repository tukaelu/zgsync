---
name: git-commit
description: |
  Analyzes git changes and creates appropriately granular commits with Conventional Commits
  format messages. Use when committing code changes, staging files, or creating structured
  commit history. Triggers on requests like "コミットして", "変更をコミット", "git commit して",
  "変更をまとめてコミット", "commit changes", "commit this", "create a commit", "stage and commit".
allowed-tools:
  - Bash(git:*)
  - Bash(go:*)
  - Bash(gofmt:*)
---

# Git Commit

Analyze changed files and create commits with appropriate granularity.

## Rules

- Do not run on the `main` branch (show a warning and abort; guide the user to create a new branch if needed)
- 1 commit = 1 logical change
- Write commit messages in Conventional Commits format

### Conventional Commits Format

Commit message format:

```
<type>(<scope>): <subject>

<body>
```

- `<type>` must be one of the following:

| type       | purpose                                         |
|------------|-------------------------------------------------|
| `feat`     | A new feature                                   |
| `fix`      | A bug fix                                       |
| `docs`     | Documentation changes                           |
| `style`    | Code style changes (formatting, etc.)           |
| `refactor` | Refactoring (no feature addition or bug fix)    |
| `perf`     | Performance improvements                        |
| `test`     | Adding or modifying tests                       |
| `build`    | Build system changes or dependency updates      |
| `ci`       | CI/CD configuration changes                     |
| `chore`    | Other changes                                   |

- Write `<subject>` and `<body>` in English, clearly describing the change

### Commit Message Examples

**Simple change:**
```
fix(zendesk): fix error when locale is not set in Translation fetch
```

**Detailed change:**
```
feat(converter): add support for Pandoc-style div block syntax

- support `:::{.class}` fence block notation
- add parsing for heading attributes `## Title {#id}`
```

**Dependency update:**
```
chore(deps): update goldmark to v1.7.13
```

**Test addition:**
```
test(cli): add frontmatter parse error handling tests for push command
```

## Workflow

### Step 1. Check branch

- Run `git branch` to confirm the current branch
- If on `main`, show a warning, abort, and guide the user to create a new branch

### Step 2. Review changes

- Run `git status` to list changed files
- Run `git diff --stat` to get change statistics
- Run `git diff` to review the diff

### Step 3. Format and lint

- Run only if `.go` files are included in the changes
- Apply formatting with `gofmt -w {file_path}`
- Run `go vet ./...` for static analysis; fix any issues before proceeding to commit

### Step 4. Group changes

**Get user approval before finalizing the grouping and removing any unnecessary code.**

- Classify changes logically (new feature, bug fix, refactoring, style fix, etc.)
- Group related files together
- Remove unnecessary debug code or comments if present

**Go-specific grouping guidelines:**
- Changes to `go.mod` / `go.sum` should be a separate commit with type `build`
- Changes to `*_test.go` files only should be separated with type `test`

### Step 5. Commit each group

- Stage files for each group with `git add`
- Write a commit message in Conventional Commits format
- Run `git commit`
- Repeat until all groups are committed

### Step 6. Verify commits

- Run `git log --oneline -5` to review the commit history
- Confirm that commit messages reflect the intended changes
