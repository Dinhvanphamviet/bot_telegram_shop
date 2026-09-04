package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set in environment or .env")
	}

	log.Printf("Connecting to Neon PostgreSQL...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse DATABASE_URL: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to Neon PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping Neon database: %v", err)
	}
	log.Println("Connected successfully to Neon PostgreSQL!")

	// 2. Read migration file
	migrationFile := "migrations/000001_init_schema.up.sql"
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", migrationFile, err)
	}

	log.Printf("Executing migration from %s...", migrationFile)

	// 3. Execute migration
	_, err = pool.Exec(ctx, string(sqlBytes))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migration executed successfully!")

	// 4. Verify tables in database
	rows, err := pool.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name;
	`)
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			log.Fatalf("Scan table name error: %v", err)
		}
		tables = append(tables, t)
	}

	log.Printf("Tables in Neon database (%d found):\n  - %s\n", len(tables), strings.Join(tables, "\n  - "))
	fmt.Println("MIGRATION_COMPLETED_SUCCESSFULLY")
}
