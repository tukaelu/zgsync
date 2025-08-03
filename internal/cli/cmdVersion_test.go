package cli

import (
	"strings"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	zgsync "github.com/tukaelu/zgsync"
)

func TestCommandVersion_Run(t *testing.T) {
	tests := []struct {
		name           string
		expectedOutput func() string
	}{
		{
			name: "version command output",
			expectedOutput: func() string {
				return "version " + zgsync.Version + " (rev: " + zgsync.Revision + ")\n"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommandVersion{}
			
			// Use testhelper to capture stdout safely
			output, err := testhelper.CaptureStdout(t, func() error {
				return cmd.Run()
			})

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			expectedOutput := tt.expectedOutput()
			if output != expectedOutput {
				t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
			}
		})
	}
}

func TestCommandVersion_OutputFormat(t *testing.T) {
	cmd := &CommandVersion{}
	
	// Use testhelper to capture stdout safely
	output, err := testhelper.CaptureStdout(t, func() error {
		return cmd.Run()
	})

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Check output format
	if !strings.HasPrefix(output, "version ") {
		t.Errorf("Expected output to start with 'version ', got '%s'", output)
	}

	if !strings.Contains(output, "(rev: ") {
		t.Errorf("Expected output to contain '(rev: ', got '%s'", output)
	}

	if !strings.HasSuffix(output, ")\n") {
		t.Errorf("Expected output to end with ')\\n', got '%s'", output)
	}

	// Check that version and revision are not empty
	parts := strings.Split(output, " ")
	if len(parts) < 3 {
		t.Errorf("Expected at least 3 parts in output, got %d", len(parts))
	}

	version := parts[1]
	if version == "" {
		t.Error("Version should not be empty")
	}

	// Extract revision from "(rev: revision)"
	revisionPart := strings.Join(parts[2:], " ")
	if !strings.HasPrefix(revisionPart, "(rev: ") {
		t.Errorf("Expected revision part to start with '(rev: ', got '%s'", revisionPart)
	}

	revision := strings.TrimPrefix(revisionPart, "(rev: ")
	revision = strings.TrimSuffix(revision, ")\n")
	if revision == "" {
		t.Error("Revision should not be empty")
	}
}

func TestCommandVersion_NoSideEffects(t *testing.T) {
	// Test that running the command multiple times produces the same output
	outputs := make([]string, 2)
	
	for i := 0; i < 2; i++ {
		cmd := &CommandVersion{}
		
		// Use testhelper to capture stdout safely
		output, err := testhelper.CaptureStdout(t, func() error {
			return cmd.Run()
		})
		
		outputs[i] = output

		if err != nil {
			t.Errorf("Run %d: Expected no error but got: %v", i+1, err)
		}
	}

	if outputs[0] != outputs[1] {
		t.Errorf("Expected consistent output, got '%s' and '%s'", outputs[0], outputs[1])
	}
}

func TestCommandVersion_Parallel(t *testing.T) {
	// Test that the command works correctly when run in parallel
	t.Parallel()
	
	cmd := &CommandVersion{}
	
	// Use testhelper to capture stdout safely
	output, err := testhelper.CaptureStdout(t, func() error {
		return cmd.Run()
	})

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	expectedOutput := "version " + zgsync.Version + " (rev: " + zgsync.Revision + ")\n"
	if output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
	}
}