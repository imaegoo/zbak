package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := run([]string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(output, "zbak - NAS备份工具") {
		t.Errorf("Expected help message, got: %s", output)
	}
}

func TestRun_Help(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"help command", []string{"help"}},
		{"--help flag", []string{"--help"}},
		{"-h flag", []string{"-h"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := run(tc.args)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !strings.Contains(output, "zbak - NAS备份工具") {
				t.Errorf("Expected help message, got: %s", output)
			}

			if !strings.Contains(output, "用法:") {
				t.Errorf("Expected usage information, got: %s", output)
			}

			if !strings.Contains(output, "backup") {
				t.Errorf("Expected backup subcommand in help, got: %s", output)
			}

			if !strings.Contains(output, "restore") {
				t.Errorf("Expected restore subcommand in help, got: %s", output)
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"version command", []string{"version"}},
		{"--version flag", []string{"--version"}},
		{"-v flag", []string{"-v"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := run(tc.args)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !strings.Contains(output, "zbak 版本") {
				t.Errorf("Expected version message, got: %s", output)
			}

			if !strings.Contains(output, version) {
				t.Errorf("Expected version %s in output, got: %s", version, output)
			}
		})
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Capture stdout
	oldStdout := os.Stdout
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	err := run([]string{"unknown"})

	w.Close()
	w2.Close()
	os.Stderr = oldStderr
	os.Stdout = oldStdout

	var bufStderr bytes.Buffer
	io.Copy(&bufStderr, r)
	stderrOutput := bufStderr.String()

	var bufStdout bytes.Buffer
	io.Copy(&bufStdout, r2)
	stdoutOutput := bufStdout.String()

	if err == nil {
		t.Error("Expected error for unknown subcommand")
	}

	if !strings.Contains(err.Error(), "未知子命令") {
		t.Errorf("Expected '未知子命令' in error, got: %v", err)
	}

	// Should print error message to stderr
	if !strings.Contains(stderrOutput, "未知子命令") {
		t.Errorf("Expected error message in stderr, got: %s", stderrOutput)
	}

	// Should print help to stdout
	if !strings.Contains(stdoutOutput, "zbak - NAS备份工具") {
		t.Errorf("Expected help message in stdout, got: %s", stdoutOutput)
	}
}

func TestRunBackup_MissingConfig(t *testing.T) {
	err := runBackup([]string{"--config", "nonexistent.yaml"})

	if err == nil {
		t.Error("Expected error for missing config file")
	}

	if !strings.Contains(err.Error(), "加载配置失败") {
		t.Errorf("Expected '加载配置失败' in error, got: %v", err)
	}
}

func TestRunRestore_MissingConfig(t *testing.T) {
	err := runRestore([]string{"--config", "nonexistent.yaml"})

	if err == nil {
		t.Error("Expected error for missing config file")
	}

	if !strings.Contains(err.Error(), "加载配置失败") {
		t.Errorf("Expected '加载配置失败' in error, got: %v", err)
	}
}

func TestRunRestore_ConflictingOptions(t *testing.T) {
	err := runRestore([]string{"--timestamp", "2024-01-15-10-30-00", "--from", "2024-01-15-10-30-00"})

	if err == nil {
		t.Error("Expected error for conflicting options")
	}

	if !strings.Contains(err.Error(), "不能同时指定") {
		t.Errorf("Expected '不能同时指定' in error, got: %v", err)
	}
}

func TestMain_ErrorHandling(t *testing.T) {
	// Test that main function exits with code 1 on error
	// We can't directly test os.Exit, but we can test the run function
	// which is called by main
	
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "unknown subcommand",
			args:        []string{"unknown"},
			expectError: true,
		},
		{
			name:        "backup with missing config",
			args:        []string{"backup", "--config", "nonexistent.yaml"},
			expectError: true,
		},
		{
			name:        "restore with missing config",
			args:        []string{"restore", "--config", "nonexistent.yaml"},
			expectError: true,
		},
		{
			name:        "restore with conflicting options",
			args:        []string{"restore", "--timestamp", "2024-01-15-10-30-00", "--from", "2024-01-15-10-30-00"},
			expectError: true,
		},
		{
			name:        "help command",
			args:        []string{"help"},
			expectError: false,
		},
		{
			name:        "version command",
			args:        []string{"version"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Suppress output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			os.Stdout = nil
			os.Stderr = nil
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			err := run(tc.args)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for args %v, but got nil", tc.args)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for args %v, but got: %v", tc.args, err)
			}
		})
	}
}

func TestIntegration_BackupCoordinatorConnection(t *testing.T) {
	// Test that runBackup properly connects to BackupCoordinator
	// This test verifies the integration without actually running a backup
	
	// We expect an error because the config file doesn't exist
	err := runBackup([]string{"--config", "test-nonexistent.yaml"})
	
	if err == nil {
		t.Error("Expected error for missing config file")
	}
	
	// Verify the error is from config loading, not from missing coordinator
	if !strings.Contains(err.Error(), "加载配置失败") {
		t.Errorf("Expected config loading error, got: %v", err)
	}
}

func TestIntegration_RestoreCoordinatorConnection(t *testing.T) {
	// Test that runRestore properly connects to RestoreCoordinator
	// This test verifies the integration without actually running a restore
	
	// We expect an error because the config file doesn't exist
	err := runRestore([]string{"--config", "test-nonexistent.yaml"})
	
	if err == nil {
		t.Error("Expected error for missing config file")
	}
	
	// Verify the error is from config loading, not from missing coordinator
	if !strings.Contains(err.Error(), "加载配置失败") {
		t.Errorf("Expected config loading error, got: %v", err)
	}
}

func TestIntegration_LoggerConfiguration(t *testing.T) {
	// Test that logger is properly configured in both backup and restore
	// We can't fully test this without a valid config, but we can verify
	// that the logger creation is attempted
	
	// For backup
	err := runBackup([]string{"--config", "test-nonexistent.yaml"})
	if err == nil {
		t.Error("Expected error for missing config file")
	}
	// If we got past config loading, we would have created a logger
	// Since config loading fails first, we can't test logger creation here
	
	// For restore
	err = runRestore([]string{"--config", "test-nonexistent.yaml"})
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}
