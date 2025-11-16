package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

// NewDB creates a new database connection and returns sqlc queries
func NewDB(dbPath string) (*sqlc.Queries, *sql.DB, error) {
	// Open database connection
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	// Set pragmas for better performance
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA cache_size = -2000",
		"PRAGMA temp_store = MEMORY",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	return sqlc.New(db), db, nil
}