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
