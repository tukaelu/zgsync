---
name: add-cli-command
description: |
  Adds a new CLI command to the zgsync project following the Kong framework pattern.
  Guides through gathering command name, purpose, and flags, generating cmdXxx.go,
  registering in cli.go, creating test file scaffolding, and verifying with make test.
  Triggers on requests like "add command", "add cli command", "implement command",
  "new command", "create command".
allowed-tools:
  - Bash(go:*)
  - Bash(make:*)
  - Edit
  - Read
  - Task
  - Write
---

# Add CLI Command

Add a new CLI command to zgsync following the Kong framework pattern.

## Workflow

### Step 1. Gather command information

Confirm the following:

1. **Command name** (e.g. `sync`, `validate`)
2. **Purpose and overview** (what the command does)
3. **Flag list** (name, type, required or optional, default value)
4. **Zendesk API methods to use** (e.g. `CreateArticle`, `ShowTranslation`)

Ask the user if anything is unclear.

### Step 2. Fetch API specification

Invoke the `zendesk-help-center-researcher` agent via the Task tool for the Zendesk API methods confirmed in Step 1.

```
Task tool:
  subagent_type: zendesk-help-center-researcher
  prompt: "Investigate the API specification for [operation confirmed in Step 1]"
```

Use the returned JSON array for the implementation in Step 3.
Confirm all of the following fields:
- `method` / `endpoint`: HTTP method and endpoint URL
- `path_parameters` / `query_parameters`: Parameter names, types, and required/optional
- `request_body`: Wrapper key and field list for the request body
- `response`: Wrapper key and field list for the response
- `notes`: Special notes such as permissions and rate limits

If the agent returns JSON containing an `error` field, report it to the user and prompt them
to manually verify the API specification before proceeding to the next step.

### Step 3. Generate cmdXxx.go

Create `internal/cli/cmd{Name}.go`.

Refer to the implementation template and Kong tag reference in [references/templates.md](references/templates.md).
Use the API specification from Step 2 to determine the correct endpoint path, parameters, and response field names.

**Naming conventions:**
- File name: `cmdXxx.go` (e.g. `cmdSync.go`)
- Type name: `CommandXxx` (e.g. `CommandSync`)
- If a Zendesk API client is required, instantiate it in `AfterApply`

### Step 4. Register in cli.go

Add a field to the `cli` struct in `internal/cli/cli.go`:

```go
type cli struct {
    Global
    // ...existing commands...
    Xxx CommandXxx `cmd:"xxx" help:"Command description."`
}
```

### Step 5. Generate test file

Create `internal/cli/cmd{Name}_test.go`.

Refer to the test template and testhelper reference in [references/templates.md](references/templates.md).

Minimum test case structure:
- Happy path (primary use cases)
- Error cases (API errors, validation errors)
- `TestCommand{Name}_AfterApply` (client initialization check)

### Step 6. Verify

```bash
make test
```

Confirm tests pass. If they fail, analyze the errors and fix them.

## Rules

- `AfterApply` only calls `zendesk.NewClient`; do not include business logic
- Always tag internal fields (`client`, `converter`) with `kong:"-"`
- Wrap and return errors with `fmt.Errorf("...: %w", err)`
- Use `testhelper.MockZendeskClient` in tests; do not call the real API
