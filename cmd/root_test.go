package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootCmd(t *testing.T) {

	if rootCmd.Use != "db-migrator" {
		t.Errorf("Expected root command use to be 'db-migrator', got %s", rootCmd.Use)
	}

	if !strings.Contains(rootCmd.Short, "struct-driven database migration tool") {
		t.Errorf("Expected short description to mention struct-driven database migration tool")
	}
}

func TestRootCmd_Help(t *testing.T) {

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute help command: %v", err)
	}

	output := buf.String()

	expectedCommands := []string{
		"migrate",
		"generate",
		"verify",
		"create",
		"introspect",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected help to contain command %q, but it didn't", cmd)
		}
	}
}

func TestExecute(t *testing.T) {

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"db-migrator", "--help"}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	Execute()
}

func TestSubcommands(t *testing.T) {

	commands := rootCmd.Commands()

	expectedCmds := map[string]bool{
		"migrate":    false,
		"generate":   false,
		"verify":     false,
		"create":     false,
		"version":    false,
		"introspect": false,
	}

	for _, cmd := range commands {

		cmdName := cmd.Use
		if spaceIdx := strings.Index(cmdName, " "); spaceIdx > 0 {
			cmdName = cmdName[:spaceIdx]
		}
		if _, ok := expectedCmds[cmdName]; ok {
			expectedCmds[cmdName] = true
		}
	}

	for cmdName, found := range expectedCmds {
		if !found {
			t.Errorf("Expected command %s to be registered, but it wasn't", cmdName)
		}
	}
}
