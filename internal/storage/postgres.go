package storage

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/quiby-ai/review-preprocessor/config"
)

type PostgresConfig = config.PostgresConfig

func MustInitPostgres(cfg PostgresConfig) *sql.DB {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		log.Fatalf("postgres open: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("postgres ping: %v", err)
	}
	if err := migrateClean(db); err != nil {
		log.Fatalf("migrate clean: %v", err)
	}
	return db
}

func migrateClean(db *sql.DB) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS clean_reviews (
		id TEXT PRIMARY KEY,
		app_id TEXT NOT NULL,
		country VARCHAR(2) NOT NULL,
		rating SMALLINT NOT NULL,
		title TEXT NOT NULL,
		content_clean TEXT NOT NULL,
		language VARCHAR(2),
		reviewed_at TIMESTAMPTZ NOT NULL,
		response_date TIMESTAMPTZ,
		response_content_clean TEXT,
		processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	_, err := db.Exec(schema)
	return err
}
