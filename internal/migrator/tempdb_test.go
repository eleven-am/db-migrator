package migrator

import (
	"strings"
	"testing"
)

func TestTempDBManager_BuildTempDBURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		tempDBName  string
		expectedURL string
	}{
		{
			name:        "basic URL with database",
			baseURL:     "postgres://user:pass@localhost:5432/maindb",
			tempDBName:  "tempdb",
			expectedURL: "postgres://user:pass@localhost:5432/tempdb",
		},
		{
			name:        "URL with query parameters",
			baseURL:     "postgres://user:pass@localhost:5432/maindb?sslmode=disable",
			tempDBName:  "tempdb",
			expectedURL: "postgres://user:pass@localhost:5432/tempdb?sslmode=disable",
		},
		{
			name:        "URL with multiple query parameters",
			baseURL:     "postgres://user:pass@localhost:5432/maindb?sslmode=disable&connect_timeout=10",
			tempDBName:  "tempdb",
			expectedURL: "postgres://user:pass@localhost:5432/tempdb?sslmode=disable&connect_timeout=10",
		},
		{
			name:        "URL with special characters in password",
			baseURL:     "postgres://user:p%40ss%23word@localhost:5432/maindb?sslmode=disable",
			tempDBName:  "tempdb",
			expectedURL: "postgres://user:p%40ss%23word@localhost:5432/tempdb?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DBConfig{
				URL: tt.baseURL,
			}
			manager := NewTempDBManager(config)

			result := manager.buildTempDBURL(tt.tempDBName)
			if result != tt.expectedURL {
				t.Errorf("buildTempDBURL(%q) = %q, want %q", tt.tempDBName, result, tt.expectedURL)
			}
		})
	}
}

func TestTempDBManager_NewTempDBManager(t *testing.T) {
	config := &DBConfig{
		URL: "postgres://user:pass@localhost:5432/testdb",
	}

	manager := NewTempDBManager(config)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.baseConfig != config {
		t.Error("Expected base config to be set")
	}
}

func TestTempDBManager_URLParsing(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		tempDBName  string
		shouldMatch bool
	}{
		{
			name:        "URL without database path",
			baseURL:     "postgres://user:pass@localhost:5432",
			tempDBName:  "tempdb",
			shouldMatch: true,
		},
		{
			name:        "URL with trailing slash",
			baseURL:     "postgres://user:pass@localhost:5432/",
			tempDBName:  "tempdb",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DBConfig{
				URL: tt.baseURL,
			}
			manager := NewTempDBManager(config)

			result := manager.buildTempDBURL(tt.tempDBName)

			if tt.shouldMatch {
				if !strings.Contains(result, tt.tempDBName) {
					t.Errorf("Expected URL to contain temp database name %q, got %q", tt.tempDBName, result)
				}
			}
		})
	}
}
