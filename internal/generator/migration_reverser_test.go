package generator

import (
	"strings"
	"testing"
)

func TestMigrationReverser_ReverseSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
		wantErr  bool
	}{
		{
			name:     "CREATE TABLE reversal",
			sql:      "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255))",
			expected: "DROP TABLE IF EXISTS users CASCADE",
		},
		{
			name:     "CREATE TABLE IF NOT EXISTS reversal",
			sql:      "CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY)",
			expected: "DROP TABLE IF EXISTS users CASCADE",
		},
		{
			name:     "ALTER TABLE ADD COLUMN reversal",
			sql:      "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			expected: "ALTER TABLE users DROP COLUMN IF EXISTS email",
		},
		{
			name:     "CREATE INDEX reversal",
			sql:      "CREATE INDEX idx_users_email ON users(email)",
			expected: "DROP INDEX IF EXISTS idx_users_email",
		},
		{
			name:     "CREATE UNIQUE INDEX reversal",
			sql:      "CREATE UNIQUE INDEX idx_users_username ON users(username)",
			expected: "DROP INDEX IF EXISTS idx_users_username",
		},
		{
			name:     "CREATE INDEX CONCURRENTLY reversal",
			sql:      "CREATE INDEX CONCURRENTLY idx_users_created ON users(created_at)",
			expected: "DROP INDEX IF EXISTS idx_users_created",
		},
		{
			name:     "ALTER TABLE ADD CONSTRAINT reversal",
			sql:      "ALTER TABLE users ADD CONSTRAINT chk_age CHECK (age >= 18)",
			expected: "ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_age",
		},
		{
			name:     "ALTER TABLE RENAME COLUMN reversal",
			sql:      "ALTER TABLE users RENAME COLUMN name TO full_name",
			expected: "ALTER TABLE users RENAME COLUMN full_name TO name",
		},
		{
			name:     "ALTER TABLE RENAME TO reversal",
			sql:      "ALTER TABLE users RENAME TO customers",
			expected: "ALTER TABLE customers RENAME TO users",
		},
		{
			name:     "CREATE SEQUENCE reversal",
			sql:      "CREATE SEQUENCE user_id_seq",
			expected: "DROP SEQUENCE IF EXISTS user_id_seq CASCADE",
		},
		{
			name:     "CREATE TYPE reversal",
			sql:      "CREATE TYPE user_role AS ENUM ('admin', 'user', 'guest')",
			expected: "DROP TYPE IF EXISTS user_role CASCADE",
		},
		{
			name:     "CREATE FUNCTION reversal",
			sql:      "CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$ SELECT COUNT(*) FROM users $$ LANGUAGE SQL",
			expected: "DROP FUNCTION IF EXISTS get_user_count CASCADE",
		},
		{
			name:     "CREATE OR REPLACE FUNCTION reversal",
			sql:      "CREATE OR REPLACE FUNCTION get_active_users() RETURNS SETOF users AS $$ SELECT * FROM users WHERE active = true $$ LANGUAGE SQL",
			expected: "DROP FUNCTION IF EXISTS get_active_users CASCADE",
		},
		{
			name:     "CREATE TRIGGER reversal",
			sql:      "CREATE TRIGGER update_timestamp BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_modified_column()",
			expected: "DROP TRIGGER IF EXISTS update_timestamp ON users",
		},
		{
			name:     "DROP TABLE returns warning",
			sql:      "DROP TABLE old_users",
			expected: "-- WARNING: Cannot reverse DROP TABLE without original schema. Backup required for restoration",
		},
		{
			name:     "DROP COLUMN returns warning",
			sql:      "ALTER TABLE users DROP COLUMN legacy_field",
			expected: "-- WARNING: Cannot reverse DROP COLUMN without original column definition. Backup required for restoration",
		},
		{
			name:     "COMMENT statement returns empty",
			sql:      "COMMENT ON TABLE users IS 'User information'",
			expected: "",
		},
		{
			name:     "Schema-qualified table name",
			sql:      "CREATE TABLE public.users (id SERIAL)",
			expected: "DROP TABLE IF EXISTS public.users CASCADE",
		},
		{
			name:     "Schema-qualified index name",
			sql:      "CREATE INDEX public.idx_users_id ON public.users(id)",
			expected: "DROP INDEX IF EXISTS public.idx_users_id",
		},
	}

	reverser := NewMigrationReverser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reverser.ReverseSQL(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReverseSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Normalize whitespace for comparison
			gotNormalized := strings.TrimSpace(got)
			expectedNormalized := strings.TrimSpace(tt.expected)

			if gotNormalized != expectedNormalized {
				t.Errorf("ReverseSQL() mismatch:\nGot:      %s\nExpected: %s", gotNormalized, expectedNormalized)
			}
		})
	}
}

func TestMigrationReverser_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		validate func(t *testing.T, result string, err error)
	}{
		{
			name: "Unhandled ALTER TABLE operation",
			sql:  "ALTER TABLE users ALTER COLUMN age SET DEFAULT 0",
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(result, "WARNING") && !strings.Contains(result, "Cannot automatically reverse ALTER COLUMN") {
					t.Errorf("Expected warning about ALTER COLUMN, got: %s", result)
				}
			},
		},
		{
			name: "Unknown statement type",
			sql:  "GRANT SELECT ON users TO readonly_user",
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(result, "WARNING") && !strings.Contains(result, "Unable to automatically reverse") {
					t.Errorf("Expected warning about unknown statement, got: %s", result)
				}
			},
		},
		{
			name: "Complex function with parameters",
			sql:  "CREATE FUNCTION add_user(name VARCHAR, email VARCHAR) RETURNS VOID AS $$ BEGIN INSERT INTO users (name, email) VALUES (name, email); END; $$ LANGUAGE plpgsql",
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(result, "DROP FUNCTION IF EXISTS add_user") {
					t.Errorf("Expected DROP FUNCTION statement, got: %s", result)
				}
			},
		},
	}

	reverser := NewMigrationReverser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reverser.ReverseSQL(tt.sql)
			tt.validate(t, got, err)
		})
	}
}
