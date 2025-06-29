package cmd

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ensureDBExists checks if the database exists and creates it if not
func ensureDBExists(dsn string) error {
	dbName, adminDSN, err := parseAndModifyDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()

	var exists bool
	err = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if exists {
		fmt.Printf("Database \"%s\" already exists.\n", dbName)
		return nil
	}

	fmt.Printf("Database \"%s\" does not exist. Creating...\n", dbName)
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(dbName)))
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	fmt.Printf("Database \"%s\" created successfully.\n", dbName)
	return nil
}

// parseAndModifyDSN extracts the database name and returns an admin DSN
func parseAndModifyDSN(dsn string) (string, string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse URL: %w", err)
		}

		dbName := strings.TrimPrefix(u.Path, "/")
		if dbName == "" {
			return "", "", fmt.Errorf("no database name in URL")
		}

		u.Path = "/postgres"
		return dbName, u.String(), nil
	}

	params := make(map[string]string)
	for _, part := range strings.Fields(dsn) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		params[kv[0]] = kv[1]
	}

	dbName, ok := params["dbname"]
	if !ok || dbName == "" {
		return "", "", fmt.Errorf("no dbname found in DSN")
	}

	params["dbname"] = "postgres"
	var parts []string
	for k, v := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return dbName, strings.Join(parts, " "), nil
}

// quoteIdentifier quotes a PostgreSQL identifier to prevent SQL injection
func quoteIdentifier(name string) string {
	return `"` + doubleQuotes(name) + `"`
}

// doubleQuotes escapes quotes in identifiers
func doubleQuotes(s string) string {
	result := ""
	for _, r := range s {
		if r == '"' {
			result += `""`
		} else {
			result += string(r)
		}
	}
	return result
}
