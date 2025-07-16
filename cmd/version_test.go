package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	// Test that version command exists
	cmd := versionCmd
	if cmd == nil {
		t.Fatal("Version command is nil")
	}

	if cmd.Use != "version" {
		t.Errorf("Expected command use to be 'version', got %s", cmd.Use)
	}

	if !strings.Contains(cmd.Short, "version information") {
		t.Error("Expected short description to mention version information")
	}
}

func TestVersionCmd_Run(t *testing.T) {
	// Just test that the Run function doesn't panic
	// The actual output goes to stdout which is harder to capture in this setup
	versionCmd.Run(versionCmd, []string{})

	// If we get here without panic, the test passes
}

func TestVersionInfo(t *testing.T) {
	// Test that version variables are set
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Test runtime information
	if runtime.Version() == "" {
		t.Error("Go version should not be empty")
	}

	if runtime.GOOS == "" {
		t.Error("OS should not be empty")
	}

	if runtime.GOARCH == "" {
		t.Error("Architecture should not be empty")
	}
}

func TestInit_VersionCmd(t *testing.T) {
	// Just verify init doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("init() panicked: %v", r)
			}
		}()
		// The init is called automatically when package loads
		// Just check the command is properly configured
		if versionCmd.Run == nil && versionCmd.RunE == nil {
			t.Error("Version command has no Run or RunE function")
		}
	}()
}
