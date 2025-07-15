package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create empty migration files",
	Long:  `Create empty UP and DOWN migration files with proper naming`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	timestamp := time.Now().UTC().Format("20060102150405")
	baseName := fmt.Sprintf("%s_%s", timestamp, name)

	upFile := filepath.Join(outputDir, fmt.Sprintf("%s.up.sql", baseName))
	downFile := filepath.Join(outputDir, fmt.Sprintf("%s.down.sql", baseName))

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	upContent := fmt.Sprintf("-- Migration: %s\n-- Created at: %s\n\n", name, time.Now().Format(time.RFC3339))
	downContent := upContent

	if err := ioutil.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to write UP migration: %w", err)
	}

	if err := ioutil.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to write DOWN migration: %w", err)
	}

	fmt.Printf("Created migration files:\n")
	fmt.Printf("  UP:   %s\n", upFile)
	fmt.Printf("  DOWN: %s\n", downFile)

	return nil
}

func init() {
	createCmd.Flags().StringVar(&outputDir, "output", "./migrations", "Output directory for migration files")
}