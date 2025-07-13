package migrator

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
)

// EnsureDatabaseExists creates the database if it doesn't exist
func EnsureDatabaseExists(dsn string) error {

	dbName, adminDSN, err := parseDSNForDB(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}

	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	if err := db.QueryRow(query, dbName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {

		fmt.Printf("Database '%s' does not exist. Creating...\n", dbName)

		createSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(dbName))
		if _, err := db.Exec(createSQL); err != nil {
			return fmt.Errorf("failed to create database '%s': %w", dbName, err)
		}

		fmt.Printf("Database '%s' created successfully.\n", dbName)
	}

	return nil
}

// parseDSNForDB extracts database name and returns admin DSN
func parseDSNForDB(dsn string) (dbName string, adminDSN string, err error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {

		parts := strings.Split(dsn, "/")
		if len(parts) < 4 {
			return "", "", fmt.Errorf("invalid database URL format")
		}

		dbPart := parts[len(parts)-1]
		if idx := strings.Index(dbPart, "?"); idx != -1 {
			dbName = dbPart[:idx]

			adminDSN = strings.Join(parts[:len(parts)-1], "/") + "/postgres?" + dbPart[idx+1:]
		} else {
			dbName = dbPart
			adminDSN = strings.Join(parts[:len(parts)-1], "/") + "/postgres"
		}
	} else {

		params := make(map[string]string)
		for _, kv := range strings.Fields(dsn) {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}

		dbName = params["dbname"]
		if dbName == "" {
			return "", "", fmt.Errorf("no database name found in DSN")
		}

		adminParts := make([]string, 0)
		for k, v := range params {
			if k == "dbname" {
				adminParts = append(adminParts, "dbname=postgres")
			} else {
				adminParts = append(adminParts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		adminDSN = strings.Join(adminParts, " ")
	}

	return dbName, adminDSN, nil
}

// quoteIdentifier quotes a PostgreSQL identifier to prevent SQL injection
func quoteIdentifier(name string) string {

	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

// GetDatabaseURL builds a database URL from components
func GetDatabaseURL(host, port, user, password, dbname, sslmode string) string {
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, url.QueryEscape(password), host, port, dbname, sslmode)
}

// GetDatabaseDSN builds a DSN string from components
func GetDatabaseDSN(host, port, user, password, dbname, sslmode string) string {
	if sslmode == "" {
		sslmode = "disable"
	}

	parts := []string{
		fmt.Sprintf("host=%s", host),
		fmt.Sprintf("port=%s", port),
		fmt.Sprintf("user=%s", user),
	}

	if password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", password))
	}

	parts = append(parts,
		fmt.Sprintf("dbname=%s", dbname),
		fmt.Sprintf("sslmode=%s", sslmode),
	)

	return strings.Join(parts, " ")
}
