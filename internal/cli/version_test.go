package cli

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		if versionCmd.Use != "version" {
			t.Errorf("expected Use to be 'version', got %s", versionCmd.Use)
		}

		if versionCmd.Short != "Show version information" {
			t.Errorf("expected Short to be 'Show version information', got %s", versionCmd.Short)
		}

		if versionCmd.Run == nil {
			t.Error("expected Run to be set")
		}
	})

	t.Run("version output", func(t *testing.T) {
		// The version command prints to stdout, so we need to test that the Run function exists
		// and doesn't panic. The actual output goes to stdout and is hard to capture in tests.
		if versionCmd.Run == nil {
			t.Error("expected Run function to be set")
		}

		// We can test that the command runs without error
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("version command panicked: %v", r)
			}
		}()

		// Run the command
		versionCmd.Run(versionCmd, []string{})
	})
}
