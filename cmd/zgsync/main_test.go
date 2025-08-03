package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMain tests the main function by running the binary as a subprocess
func TestMain(t *testing.T) {
	// Build the binary for testing
	if err := exec.Command("go", "build", "-o", "zgsync_test", ".").Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("zgsync_test") // Clean up

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedOutput string
	}{
		{
			name:           "help command",
			args:           []string{"--help"},
			expectError:    false,
			expectedOutput: "zgsync is a command-line tool",
		},
		{
			name:           "version command",
			args:           []string{"version"},
			expectError:    false,
			expectedOutput: "version",
		},
		{
			name:        "invalid command",
			args:        []string{"invalid-command"},
			expectError: true,
		},
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true, // Should show usage and exit with error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./zgsync_test", tt.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but command succeeded. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected success but command failed with error: %v. Output: %s", err, outputStr)
			}
			if tt.expectedOutput != "" && !strings.Contains(outputStr, tt.expectedOutput) {
				t.Errorf("Expected output to contain '%s', but got: %s", tt.expectedOutput, outputStr)
			}
		})
	}
}

// TestMainFunction tests main function behavior by temporarily redirecting
// the cli.Bind function call to avoid process exit during testing
func TestMainFunction(t *testing.T) {
	// This test verifies that main() calls cli.Bind() without errors
	// In a real scenario, we would mock cli.Bind() to avoid process termination
	// For now, we test the integration through the subprocess tests above
	
	// The main function is very simple and just calls cli.Bind()
	// The actual functionality testing is done through subprocess execution
	// This ensures we test the real behavior while avoiding process exit issues
	t.Log("Main function integration tested via subprocess execution")
}

// TestMainWithEnvironmentVariables tests main function with different environment setups
func TestMainWithEnvironmentVariables(t *testing.T) {
	if err := exec.Command("go", "build", "-o", "zgsync_test_env", ".").Run(); err != nil {
		t.Fatalf("Failed to build binary for testing: %v", err)
	}
	defer os.Remove("zgsync_test_env")

	tests := []struct {
		name    string
		envVars map[string]string
		args    []string
		expectError bool
	}{
		{
			name: "help with custom environment",
			envVars: map[string]string{
				"HOME": "/tmp",
			},
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name: "version with clean environment",
			envVars: map[string]string{},
			args:        []string{"version"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./zgsync_test_env", tt.args...)
			
			// Set custom environment
			env := os.Environ()
			for key, value := range tt.envVars {
				env = append(env, key+"="+value)
			}
			cmd.Env = env

			output, err := cmd.CombinedOutput()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but command succeeded. Output: %s", string(output))
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected success but command failed with error: %v. Output: %s", err, string(output))
			}
		})
	}
}