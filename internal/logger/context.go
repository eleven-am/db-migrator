package logger

// Component-specific logger functions

// Schema returns a logger for schema generation operations
func Schema() Logger {
	return WithField("component", "schema")
}

// SQL returns a logger for SQL generation operations
func SQL() Logger {
	return WithField("component", "sql")
}

// Migration returns a logger for migration operations
func Migration() Logger {
	return WithField("component", "migration")
}

// Atlas returns a logger for Atlas operations
func Atlas() Logger {
	return WithField("component", "atlas")
}

// CLI returns a logger for CLI operations
func CLI() Logger {
	return WithField("component", "cli")
}

// DB returns a logger for database operations
func DB() Logger {
	return WithField("component", "db")
}

// ORM returns a logger for ORM generation operations
func ORM() Logger {
	return WithField("component", "orm")
}

// Parser returns a logger for parsing operations
func Parser() Logger {
	return WithField("component", "parser")
}
