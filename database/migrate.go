package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version  string
	Filename string
	SQL      string
}

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if !contains(appliedMigrations, migration.Version) {
			fmt.Printf("Running migration: %s\n", migration.Filename)

			if err := runMigration(db, migration); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", migration.Filename, err)
			}

			if err := recordMigration(db, migration.Version); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.Filename, err)
			}
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

// findMigrationsDir finds the migrations directory from current or parent directories
func findMigrationsDir() (string, error) {
	// Try current directory first
	if _, err := os.Stat("database/migrations"); err == nil {
		return "database/migrations", nil
	}

	// Try going up directories to find project root
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up the directory tree looking for database/migrations or go.mod
	for {
		// Check for database/migrations in current directory
		migrationsPath := filepath.Join(currentDir, "database", "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath, nil
		}

		// Check if we're at project root (has go.mod)
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			// We're at project root, but no migrations directory found
			return filepath.Join(currentDir, "database", "migrations"), nil
		}

		// Go up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// We've reached the root directory
			break
		}
		currentDir = parent
	}

	// Fallback to relative path
	return "database/migrations", nil
}

// loadMigrations loads all migration files from the migrations directory
func loadMigrations() ([]Migration, error) {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find migrations directory: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	var migrations []Migration
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		filename := filepath.Base(file)
		version := strings.TrimSuffix(filename, ".sql")

		migrations = append(migrations, Migration{
			Version:  version,
			Filename: filename,
			SQL:      string(content),
		})
	}

	// Sort migrations by filename/version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations returns list of already applied migration versions
func getAppliedMigrations(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT version FROM migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

// runMigration executes a single migration
func runMigration(db *sql.DB, migration Migration) error {
	// Execute the migration SQL
	_, err := db.Exec(migration.SQL)
	return err
}

// recordMigration marks a migration as applied
func recordMigration(db *sql.DB, version string) error {
	_, err := db.Exec("INSERT INTO migrations (version) VALUES (?)", version)
	return err
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
