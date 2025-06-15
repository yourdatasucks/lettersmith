package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down]")
	}

	command := os.Args[1]

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	switch command {
	case "up":
		if err := runMigrationsUp(db); err != nil {
			log.Fatal("Migration up failed:", err)
		}
	case "down":
		if err := runMigrationsDown(db); err != nil {
			log.Fatal("Migration down failed:", err)
		}
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func ensureMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)`
	_, err := db.Exec(query)
	return err
}

func runMigrationsUp(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	migrationFiles, err := getMigrationFiles()
	if err != nil {
		return err
	}

	for _, file := range migrationFiles {
		version := strings.TrimSuffix(file, ".sql")

		if appliedMigrations[version] {
			fmt.Printf("Skipping already applied migration: %s\n", version)
			continue
		}

		fmt.Printf("Applying migration: %s\n", version)

		if err := applyMigration(db, file, version); err != nil {
			return fmt.Errorf("failed to apply migration %s: %v", version, err)
		}

		if err := markMigrationApplied(db, version); err != nil {
			return fmt.Errorf("failed to mark migration as applied %s: %v", version, err)
		}
	}

	fmt.Println("All migrations applied successfully")
	return nil
}

func runMigrationsDown(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	migrationFiles, err := getMigrationFiles()
	if err != nil {
		return err
	}

	for i := len(migrationFiles) - 1; i >= 0; i-- {
		file := migrationFiles[i]
		version := strings.TrimSuffix(file, ".sql")

		if !appliedMigrations[version] {
			fmt.Printf("Skipping unapplied migration: %s\n", version)
			continue
		}

		fmt.Printf("Rolling back migration: %s\n", version)

		if err := rollbackMigration(db, version); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %v", version, err)
		}
	}

	fmt.Println("All migrations rolled back successfully")
	return nil
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func getMigrationFiles() ([]string, error) {
	var files []string

	err := filepath.WalkDir("migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			files = append(files, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func applyMigration(db *sql.DB, filename, version string) error {
	content, err := os.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return err
	}

	sqlContent := string(content)
	statements := strings.Split(sqlContent, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error executing statement: %v\nStatement: %s", err, stmt)
		}
	}

	return nil
}

func markMigrationApplied(db *sql.DB, version string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
	return err
}

func rollbackMigration(db *sql.DB, version string) error {
	_, err := db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
	return err
}
