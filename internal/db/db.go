package db

import (
	"avito-shop/internal/config"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Функция для подключения к базе данных инфорация для дебага и пинг, чтобы проверить что она жива
func Connect(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
	)
	fmt.Printf("Connecting to database: host=%s port=%s user=%s password=%s dbname=%s sslmode=disable\n",
		cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
