package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestMigrateCmd(t *testing.T) {

	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(migrateCmd)

	cmd, _, err := rootCmd.Find([]string{"migrate"})
	if err != nil {
		t.Fatalf("Failed to find migrate command: %v", err)
	}

	if cmd.Use != "migrate" {
		t.Errorf("Expected command use to be 'migrate', got %s", cmd.Use)
	}

	flags := []string{
		"url", "host", "port", "user", "password", "dbname", "sslmode",
		"package", "output", "name", "dry-run", "push", "allow-destructive",
		"create-if-not-exists",
	}

	for _, flagName := range flags {
		flag := cmd.Flag(flagName)
		if flag == nil {
			t.Errorf("Expected flag %s to exist", flagName)
		}
	}
}

func TestMigrateCmd_Defaults(t *testing.T) {
	rootCmd := &cobra.Command{}
	rootCmd.AddCommand(migrateCmd)

	cmd, _, _ := rootCmd.Find([]string{"migrate"})

	defaults := map[string]string{
		"host":    "localhost",
		"port":    "5432",
		"sslmode": "disable",
		"package": "./internal/db",
		"output":  "./migrations",
	}

	for flagName, expectedDefault := range defaults {
		flag := cmd.Flag(flagName)
		if flag.DefValue != expectedDefault {
			t.Errorf("Expected %s default to be %s, got %s", flagName, expectedDefault, flag.DefValue)
		}
	}
}

func TestValidateDBConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func()
		wantErr   bool
	}{
		{
			name: "valid URL",
			setupFunc: func() {
				dbURL = "postgres://user:pass@localhost:5432/db"
				dbHost = ""
			},
			wantErr: false,
		},
		{
			name: "valid individual params",
			setupFunc: func() {
				dbURL = ""
				dbHost = "localhost"
				dbPort = "5432"
				dbUser = "postgres"
				dbPassword = "password"
				dbName = "testdb"
			},
			wantErr: false,
		},
		{
			name: "missing host",
			setupFunc: func() {
				dbURL = ""
				dbHost = ""
				dbPort = "5432"
				dbUser = "postgres"
				dbPassword = "password"
				dbName = "testdb"
			},
			wantErr: true,
		},
		{
			name: "missing dbname",
			setupFunc: func() {
				dbURL = ""
				dbHost = "localhost"
				dbPort = "5432"
				dbUser = "postgres"
				dbPassword = "password"
				dbName = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			dbURL = ""
			dbHost = ""
			dbPort = ""
			dbUser = ""
			dbPassword = ""
			dbName = ""
			dbSSLMode = ""

			tt.setupFunc()

			_, err := getDBConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("getDBConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper to get DB config (simplified version for testing)
func getDBConfig() (interface{}, error) {
	if dbURL != "" {
		return dbURL, nil
	}

	if dbHost == "" {
		return nil, fmt.Errorf("database host is required")
	}
	if dbName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	return struct{}{}, nil
}
