
package database

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func NewPostgres(
	ctx context.Context,
	databaseURL string,
	maxConns int32,
	minConns int32,
) (*pgxpool.Pool, error) {

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	log.Println("PostgreSQL connected successfully")

	return pool, nil
}

// Run database migrations
func RunMigrations(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	if err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	files, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {

		if file.IsDir() {
			continue
		}

		name := file.Name()

		applied := false

		err := pool.QueryRow(
			ctx,
			`SELECT EXISTS (
				SELECT 1 FROM schema_migrations WHERE version = $1
			)`,
			name,
		).Scan(&applied)

		if err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		if applied {
			log.Printf("Migration already applied: %s", name)
			continue
		}

		content, err := migrationFiles.ReadFile(
			"migrations/" + name,
		)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`,
			name,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		log.Printf("Migration applied successfully: %s", name)
	}

	return nil
}
