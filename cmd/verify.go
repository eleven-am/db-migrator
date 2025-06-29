package cmd

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify database schema",
	Long:  `Verify that the database schema matches expectations`,
	RunE:  runVerify,
}

func init() {
	verifyCmd.Flags().StringVar(&dbURL, "url", "", "Database connection URL")
	verifyCmd.Flags().StringVar(&dbHost, "host", "localhost", "Database host")
	verifyCmd.Flags().StringVar(&dbPort, "port", "5432", "Database port")
	verifyCmd.Flags().StringVar(&dbUser, "user", "", "Database user")
	verifyCmd.Flags().StringVar(&dbPassword, "password", "", "Database password")
	verifyCmd.Flags().StringVar(&dbName, "dbname", "", "Database name")
	verifyCmd.Flags().StringVar(&dbSSLMode, "sslmode", "disable", "SSL mode")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Determine DSN
	var dsn string
	if dbURL != "" {
		dsn = dbURL
	} else if dbUser != "" && dbName != "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	} else {
		return fmt.Errorf("either --url or both --user and --dbname must be provided")
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Check tables exist
	tables := []string{
		"teams", "projects", "users", "pipelines", "triggers",
		"destinations", "stages", "execution_logs", "audit_logs",
		"api_keys", "oauth_tokens", "auth_credentials", "dlq_messages",
		"plans", "subscriptions",
	}

	fmt.Println("Verifying database schema...")
	fmt.Println()

	allGood := true
	for _, table := range tables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, table).Scan(&exists)
		
		if err != nil {
			fmt.Printf("ERROR: Error checking table %s: %v\n", table, err)
			allGood = false
			continue
		}

		if exists {
			// Count rows
			var count int
			db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
			fmt.Printf("OK: Table %-20s exists (rows: %d)\n", table, count)
		} else {
			fmt.Printf("MISSING: Table %-20s missing\n", table)
			allGood = false
		}
	}

	// Check gen_cuid function
	var funcExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc 
			WHERE proname = 'gen_cuid'
		)
	`).Scan(&funcExists)
	
	fmt.Println()
	if funcExists {
		fmt.Println("OK: Function gen_cuid() exists")
	} else {
		fmt.Println("MISSING: Function gen_cuid() missing")
		allGood = false
	}

	fmt.Println()
	if allGood {
		fmt.Println("Schema verification passed!")
	} else {
		fmt.Println("Schema verification failed!")
		return fmt.Errorf("schema verification failed")
	}

	return nil
}