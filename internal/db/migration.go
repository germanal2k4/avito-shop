package db

import (
	"database/sql"
	"log"

	"github.com/pressly/goose/v3"
)

func Migrate(db *sql.DB, migrationsDir string) {
	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
}
