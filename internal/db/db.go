package db

import (
	"avito-shop/internal/config"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type CoinInventoryDB interface {
	BeginTx() (*sql.Tx, error)
	GetCoinsForUpdate(tx *sql.Tx, userID int) (int, error)
	IncreaseCoins(tx *sql.Tx, userID, amount int) error
	DecreaseCoins(tx *sql.Tx, userID, amount int) error
	IncreaseItem(tx *sql.Tx, userID int, item string, delta int) error
	InsertTransaction(tx *sql.Tx, userID int, transactionType, counterparty string, amount int) error
	InsertReceivedTransaction(tx *sql.Tx, toUserID, fromUserID, amount int) error
	GetUserIDByUsernameForUpdate(tx *sql.Tx, username string) (int, error)
	GetUserCoins(userID int) (int, error)
	GetInventory(userID int) ([]InventoryItem, error)
	GetTransactions(userID int, transactionType string) ([]Transaction, error)
}
type InventoryItem struct {
	Type     string
	Quantity int
}

type Transaction struct {
	Counterparty string
	Amount       int
}

type AuthDB interface {
	GetUserAuthData(username string) (int, string, error)
}

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
