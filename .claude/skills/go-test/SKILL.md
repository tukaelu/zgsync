---
name: go-test
description: |
  Runs Go tests, analyzes failures, and proposes fixes. Use when running tests,
  investigating test failures, or checking coverage. Triggers on requests like
  "run tests", "test this", "run the tests", "check test coverage",
  "テスト実行して", "テスト失敗を確認", "テストを走らせて", "テストが失敗している".
allowed-tools:
  - Bash(go:*)
  - Bash(make:*)
  - Grep
  - Read
---

# Go Test

Run Go tests, analyze failures, and propose fixes.

## Workflow

### Step 1. Run tests

Run the full test suite:

```bash
make test
```

If a specific package is targeted, use:

```bash
go test -v ./internal/<package>/...
```

### Step 2. Evaluate results

**All tests pass** → Show coverage summary (Step 4) and finish.

**Failures detected** → Proceed to Step 3.

### Step 3. Analyze failures

For each failing test:

1. Identify the test name and package from the output
2. Read the test file to understand the test intent
3. Read the implementation file being tested
4. Identify the root cause of the failure
5. Propose a concrete fix with a code snippet

**Common failure patterns in this project:**

| Pattern                   | Where to look                                        |
|---------------------------|------------------------------------------------------|
| Frontmatter parse error   | `internal/zendesk/article.go`, `translation.go`      |
| API mock mismatch         | `internal/zendesk/mock_*.go`, `mock_scenarios.go`    |
| CLI config error          | `internal/cli/config.go`, `cmdPush.go`, `cmdPull.go` |
| Converter output mismatch | `internal/converter/converter.go`                    |

After all failures are resolved, return to Step 1 to confirm the full suite passes, then proceed to Step 4.

### Step 4. Show coverage summary

Run with coverage and summarize:

```bash
go test -cover ./...
```

Report packages with coverage below 60% as areas that may need additional tests.

## Rules

- Fix one failing test at a time; re-run after each fix to confirm
- Do not modify test files to make tests pass unless the test itself is wrong
- Follow table-driven test patterns when adding new test cases
- Use `go test -v -run <TestName> ./internal/<package>/...` to isolate a single test
