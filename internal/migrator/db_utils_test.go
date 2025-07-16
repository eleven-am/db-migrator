package migrator

import (
	"testing"
)

func TestGetDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		user     string
		password string
		dbname   string
		sslmode  string
		expected string
	}{
		{
			name:     "basic connection",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "secret",
			dbname:   "testdb",
			sslmode:  "disable",
			expected: "postgres://postgres:secret@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "with special characters in password",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "p@ss#word",
			dbname:   "testdb",
			sslmode:  "disable",
			expected: "postgres://postgres:p%40ss%23word@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "with empty password",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "",
			dbname:   "testdb",
			sslmode:  "disable",
			expected: "postgres://postgres:@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "with require sslmode",
			host:     "remote.db.com",
			port:     "5432",
			user:     "dbuser",
			password: "dbpass",
			dbname:   "production",
			sslmode:  "require",
			expected: "postgres://dbuser:dbpass@remote.db.com:5432/production?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDatabaseURL(tt.host, tt.port, tt.user, tt.password, tt.dbname, tt.sslmode)
			if result != tt.expected {
				t.Errorf("GetDatabaseURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetDatabaseDSN(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		user     string
		password string
		dbname   string
		sslmode  string
		expected string
	}{
		{
			name:     "basic DSN",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "secret",
			dbname:   "testdb",
			sslmode:  "disable",
			expected: "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable",
		},
		{
			name:     "with empty password",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "",
			dbname:   "testdb",
			sslmode:  "disable",
			expected: "host=localhost port=5432 user=postgres dbname=testdb sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDatabaseDSN(tt.host, tt.port, tt.user, tt.password, tt.dbname, tt.sslmode)
			if result != tt.expected {
				t.Errorf("GetDatabaseDSN() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "users",
			expected: "\"users\"",
		},
		{
			name:     "identifier with spaces",
			input:    "user data",
			expected: "\"user data\"",
		},
		{
			name:     "identifier with special characters",
			input:    "user-table",
			expected: "\"user-table\"",
		},
		{
			name:     "empty identifier",
			input:    "",
			expected: "\"\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("quoteIdentifier() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseDSNForDB(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		expectedDBName string
		expectedAdmin  string
		shouldError    bool
	}{
		{
			name:           "postgres URL",
			dsn:            "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			expectedDBName: "testdb",
			expectedAdmin:  "postgres://user:pass@localhost:5432/postgres?sslmode=disable",
			shouldError:    false,
		},
		{
			name:           "postgresql URL",
			dsn:            "postgresql://user:pass@localhost:5432/testdb?sslmode=require",
			expectedDBName: "testdb",
			expectedAdmin:  "postgresql://user:pass@localhost:5432/postgres?sslmode=require",
			shouldError:    false,
		},
		{
			name:           "URL with query params",
			dsn:            "postgres://user:pass@localhost:5432/testdb?sslmode=disable&timezone=UTC",
			expectedDBName: "testdb",
			expectedAdmin:  "postgres://user:pass@localhost:5432/postgres?sslmode=disable&timezone=UTC",
			shouldError:    false,
		},
		{
			name:        "invalid URL",
			dsn:         "invalid-url",
			shouldError: true,
		},
		{
			name:        "unsupported scheme",
			dsn:         "mysql://user:pass@localhost:3306/testdb",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbName, adminDSN, err := parseDSNForDB(tt.dsn)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if dbName != tt.expectedDBName {
				t.Errorf("Expected dbName %q, got %q", tt.expectedDBName, dbName)
			}
			if adminDSN != tt.expectedAdmin {
				t.Errorf("Expected adminDSN %q, got %q", tt.expectedAdmin, adminDSN)
			}
		})
	}
}
