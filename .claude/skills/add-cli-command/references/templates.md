# Command Implementation Templates

## Table of Contents

- [cmdXxx.go Implementation Template](#internalclicmdxxxgo)
- [Kong Tag Reference](#kong-tag-reference)
- [cmdXxx_test.go Test Template](#internalclicmdxxxtestgo)
- [testhelper Helpers Reference](#testhelper-helpers-reference)

---

## internal/cli/cmdXxx.go

```go
package cli

import (
	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandXxx struct {
	// Flag examples (defined with Kong tags)
	// SomeFlag string `name:"some-flag" short:"f" help:"Description."`
	client zendesk.Client `kong:"-"`
}

func (c *CommandXxx) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *CommandXxx) Run(g *Global) error {
	// implementation
	return nil
}
```

**Kong Tag Reference:**

| Tag                    | Purpose                                        | Example                    |
|------------------------|------------------------------------------------|----------------------------|
| `name:"xxx"`           | Flag name (`--xxx`)                            | `name:"locale"`            |
| `short:"x"`            | Short flag (`-x`)                              | `short:"l"`                |
| `help:"..."`           | Help text                                      | `help:"Specify the locale."` |
| `required:""`          | Required flag                                  | `required:""`              |
| `arg:""`               | Positional argument                            | `arg:""`                   |
| `type:"existingfile"`  | Accept only existing files                     | `type:"existingfile"`      |
| `kong:"-"`             | Ignored by Kong (internal field)               | `kong:"-"`                 |

---

## internal/cli/cmdXxx_test.go

```go
package cli

import (
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
)

func TestCommandXxx_Run(t *testing.T) {
	tests := []struct {
		name        string
		cmd         CommandXxx
		expectError bool
		mockSetup   func(*testhelper.MockZendeskClient)
	}{
		{
			name:        "success: ...",
			cmd:         CommandXxx{},
			expectError: false,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				// mock.SomeFuncFunc = func(...) (...) { ... }
			},
		},
		{
			name:        "error: API error",
			cmd:         CommandXxx{},
			expectError: true,
			mockSetup: func(mock *testhelper.MockZendeskClient) {
				// mock.SomeFuncFunc = func(...) (...) { return "", fmt.Errorf("api error") }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testhelper.MockZendeskClient{}
			tt.mockSetup(mockClient)

			cmd := tt.cmd
			cmd.client = mockClient

			global := &Global{
				Config: Config{
					DefaultLocale:            testhelper.TestLocales.Japanese,
					DefaultPermissionGroupID: testhelper.TestPermissionGroupID,
					ContentsDir:              t.TempDir(),
				},
			}

			err := cmd.Run(global)
			if tt.expectError && err == nil {
				t.Errorf("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCommandXxx_AfterApply(t *testing.T) {
	global := &Global{
		Config: Config{
			Subdomain: "test",
			Email:     "test@example.com",
			Token:     "token",
		},
	}

	cmd := &CommandXxx{}
	if err := cmd.AfterApply(global); err != nil {
		t.Errorf("AfterApply() failed: %v", err)
	}
	if cmd.client == nil {
		t.Error("client is not initialized")
	}
}
```

---

## testhelper Helpers Reference

| Constant / Function                                                       | Description                                      |
|---------------------------------------------------------------------------|--------------------------------------------------|
| `testhelper.TestSectionID`                                                | Section ID for testing                           |
| `testhelper.TestPermissionGroupID`                                        | Permission group ID for testing                  |
| `testhelper.TestUserSegmentID`                                            | User segment ID for testing                      |
| `testhelper.TestLocales.Japanese`                                         | `"ja"`                                           |
| `testhelper.TestLocales.English`                                          | `"en_us"`                                        |
| `testhelper.IntPtr(n)`                                                    | Returns a pointer to an `*int` value             |
| `testhelper.CreateDefaultArticleResponse(id, sectionID)`                 | Generates an Article JSON response               |
| `testhelper.CreateDefaultTranslationResponse(id, articleID, locale)`     | Generates a Translation JSON response            |
| `testhelper.MockZendeskClient`                                            | Mock Zendesk client                              |
