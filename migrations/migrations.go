package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB) error {
	log.Println("Starting database migrations...")

	// Create migrations tracking table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get list of applied migrations
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}

	// Read migration files
	migrations, err := readMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to read migration files: %v", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if _, applied := appliedMigrations[migration.Version]; applied {
			log.Printf("Migration %03d_%s already applied, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration %03d_%s...", migration.Version, migration.Name)
		if err := applyMigration(db, migration); err != nil {
			return fmt.Errorf("failed to apply migration %03d_%s: %v", migration.Version, migration.Name, err)
		}
		log.Printf("Successfully applied migration %03d_%s", migration.Version, migration.Name)
	}

	log.Println("All migrations completed successfully")
	return nil
}

// createMigrationsTable creates the schema_migrations table
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`

	_, err := db.Exec(query)
	return err
}

// getAppliedMigrations returns a map of applied migration versions
func getAppliedMigrations(db *sql.DB) (map[int]bool, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// readMigrationFiles reads all migration files from the migrations directory
func readMigrationFiles() ([]Migration, error) {
	migrationsDir := "migrations"

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory not found")
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Parse migration version from filename (e.g., "001_add_user_game_fields.sql")
		var version int
		var name string
		_, err := fmt.Sscanf(file.Name(), "%d_%s", &version, &name)
		if err != nil {
			log.Printf("Warning: Skipping file with invalid format: %s", file.Name())
			continue
		}

		// Remove .sql extension from name
		name = strings.TrimSuffix(name, ".sql")

		// Read file contents
		filePath := filepath.Join(migrationsDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// applyMigration executes a migration and records it in schema_migrations
func applyMigration(db *sql.DB, migration Migration) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return err
	}

	// Record migration in schema_migrations table
	recordQuery := `
		INSERT INTO schema_migrations (version, name, applied_at)
		VALUES ($1, $2, NOW())`

	if _, err := tx.Exec(recordQuery, migration.Version, migration.Name); err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}
