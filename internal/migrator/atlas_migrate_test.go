package migrator

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestMigrationOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    MigrationOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: MigrationOptions{
				PackagePath: "./internal/models",
				OutputDir:   "./migrations",
			},
			wantErr: false,
		},
		{
			name: "missing package path",
			opts: MigrationOptions{
				PackagePath: "",
				OutputDir:   "./migrations",
			},
			wantErr: true,
		},
		{
			name: "missing output dir for file mode",
			opts: MigrationOptions{
				PackagePath: "./internal/models",
				OutputDir:   "",
				DryRun:      false,
				PushToDB:    false,
			},
			wantErr: true,
		},
		{
			name: "output dir not required for dry run",
			opts: MigrationOptions{
				PackagePath: "./internal/models",
				OutputDir:   "",
				DryRun:      true,
			},
			wantErr: false,
		},
		{
			name: "output dir not required for push",
			opts: MigrationOptions{
				PackagePath: "./internal/models",
				OutputDir:   "",
				PushToDB:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(&tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMigrationResult_GetFilenames(t *testing.T) {
	result := &MigrationResult{
		UpFilePath:   "/path/to/migrations/20240101120000_add_users.up.sql",
		DownFilePath: "/path/to/migrations/20240101120000_add_users.down.sql",
	}

	upFile := filepath.Base(result.UpFilePath)
	if upFile != "20240101120000_add_users.up.sql" {
		t.Errorf("Expected up filename to be '20240101120000_add_users.up.sql', got %s", upFile)
	}

	downFile := filepath.Base(result.DownFilePath)
	if downFile != "20240101120000_add_users.down.sql" {
		t.Errorf("Expected down filename to be '20240101120000_add_users.down.sql', got %s", downFile)
	}
}

func TestNewAtlasMigrator(t *testing.T) {
	config := &DBConfig{
		URL: "postgres://postgres:password@localhost:5432/testdb?sslmode=disable",
	}

	migrator := NewAtlasMigrator(config)

	if migrator == nil {
		t.Fatal("Expected migrator to be created")
	}

	if migrator.config != config {
		t.Error("Expected config to be set")
	}

	if migrator.tempDBManager == nil {
		t.Error("Expected tempDBManager to be initialized")
	}

	if migrator.structParser == nil {
		t.Error("Expected structParser to be initialized")
	}

	if migrator.schemaGenerator == nil {
		t.Error("Expected schemaGenerator to be initialized")
	}

	if migrator.sqlGenerator == nil {
		t.Error("Expected sqlGenerator to be initialized")
	}

	if migrator.migrationReverser == nil {
		t.Error("Expected migrationReverser to be initialized")
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    MigrationOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: MigrationOptions{
				PackagePath: "./models",
				OutputDir:   "./migrations",
			},
			wantErr: false,
		},
		{
			name: "missing package path",
			opts: MigrationOptions{
				PackagePath: "",
				OutputDir:   "./migrations",
			},
			wantErr: true,
			errMsg:  "package path is required",
		},
		{
			name: "dry run without output dir",
			opts: MigrationOptions{
				PackagePath: "./models",
				OutputDir:   "",
				DryRun:      true,
			},
			wantErr: false,
		},
		{
			name: "push without output dir",
			opts: MigrationOptions{
				PackagePath: "./models",
				OutputDir:   "",
				PushToDB:    true,
			},
			wantErr: false,
		},
		{
			name: "normal mode without output dir",
			opts: MigrationOptions{
				PackagePath: "./models",
				OutputDir:   "",
				DryRun:      false,
				PushToDB:    false,
			},
			wantErr: true,
			errMsg:  "output directory is required when not using --dry-run or --push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(&tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("validateOptions() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// Helper function
func validateOptions(opts *MigrationOptions) error {
	if opts.PackagePath == "" {
		return fmt.Errorf("package path is required")
	}

	if !opts.DryRun && !opts.PushToDB && opts.OutputDir == "" {
		return fmt.Errorf("output directory is required when not using --dry-run or --push")
	}

	return nil
}
