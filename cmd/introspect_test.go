package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestIntrospectCmd(t *testing.T) {

	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(introspectCmd)

	cmd, _, err := rootCmd.Find([]string{"introspect"})
	if err != nil {
		t.Fatalf("Failed to find introspect command: %v", err)
	}

	if cmd.Use != "introspect" {
		t.Errorf("Expected command use to be 'introspect', got %s", cmd.Use)
	}

	dbFlag := cmd.Flag("database")
	if dbFlag == nil {
		t.Error("Expected database flag to exist")
	}

	formatFlag := cmd.Flag("format")
	if formatFlag == nil {
		t.Error("Expected format flag to exist")
	}
	if formatFlag.DefValue != "markdown" {
		t.Errorf("Expected format default to be 'markdown', got %s", formatFlag.DefValue)
	}

	packageFlag := cmd.Flag("package")
	if packageFlag == nil {
		t.Error("Expected package flag to exist")
	}
	if packageFlag.DefValue != "models" {
		t.Errorf("Expected package default to be 'models', got %s", packageFlag.DefValue)
	}

	schemaFlag := cmd.Flag("schema")
	if schemaFlag == nil {
		t.Error("Expected schema flag to exist")
	}
	if schemaFlag.DefValue != "public" {
		t.Errorf("Expected schema default to be 'public', got %s", schemaFlag.DefValue)
	}
}

func TestIntrospectCmd_Help(t *testing.T) {
	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(introspectCmd)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"introspect", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute help command: %v", err)
	}

	output := buf.String()

	expectedContents := []string{
		"Inspect database schema",
		"--database",
		"--format",
		"--output",
		"--table",
		"--schema",
		"--package",
		"json, yaml, markdown, sql, dot, go",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help to contain %q, but it didn't", expected)
		}
	}
}

func TestIntrospectCmd_MissingDatabase(t *testing.T) {

	oldDBURL := introspectDBURL
	defer func() { introspectDBURL = oldDBURL }()
	introspectDBURL = ""

	testCmd := &cobra.Command{
		Use:   "introspect",
		Short: introspectCmd.Short,
		RunE:  introspectCmd.RunE,
	}

	testCmd.Flags().StringVarP(&introspectDBURL, "database", "d", "", "Database connection URL (required)")
	testCmd.MarkFlagRequired("database")

	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(testCmd)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"introspect"})
	err := rootCmd.Execute()

	if err == nil {
		t.Error("Expected error when database flag is missing")
		return
	}

	if !strings.Contains(err.Error(), "required flag") && !strings.Contains(err.Error(), "database") {
		t.Errorf("Expected error about missing database flag, got: %v", err)
	}
}
