---
name: go-lint-fix
description: |
  Runs Go code quality checks and auto-fixes formatting issues. Applies gofmt for
  formatting, go vet for static analysis, and golangci-lint for linting. Auto-fixes
  what can be fixed automatically and reports issues requiring manual attention.
  Use when checking code quality, fixing formatting, or running linters. Triggers on
  requests like "lint fix", "run linter", "fix formatting", "go vet",
  "lintして", "コードを整形", "フォーマット修正", "コード品質チェック".
allowed-tools:
  - Bash(go:*)
  - Bash(gofmt:*)
  - Bash(golangci-lint:*)
  - Bash(make:*)
  - Read
---

# Go Lint Fix

Run Go code quality checks and auto-fix formatting issues.

## Workflow

### Step 1. Format with gofmt

Apply formatting to all Go files and list modified files:

```bash
gofmt -l -w .
```

`-l` prints files that were reformatted; `-w` writes the changes in place.

### Step 2. Static analysis with go vet

```bash
go vet ./...
```

**If errors are found:** Read the relevant source file, identify the issue, fix it, then re-run `go vet ./...` to confirm it is clean before proceeding.

### Step 3. Lint with golangci-lint

```bash
make lint
```

### Step 4. Report results

Categorize issues into two groups:

**Auto-fixed:**
- List files reformatted by gofmt

**Requires manual attention:**
- List each remaining lint issue with file:line and a brief explanation
- Propose a fix for each issue

If no issues remain, confirm all checks pass.

## Rules

- Run gofmt before go vet and golangci-lint — formatting errors can mask other issues
- Do not suppress lint warnings with `//nolint` directives unless the user explicitly requests it
- If golangci-lint is not installed, guide the user to install it:
  - macOS: `brew install golangci-lint`
  - Others: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
