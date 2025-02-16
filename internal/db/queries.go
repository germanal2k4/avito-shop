package db

import (
	"database/sql"
	"fmt"
)

type coinInventoryDBImplementation struct {
	db *sql.DB
}

func NewCoinInventoryDB(dbConn *sql.DB) CoinInventoryDB {
	return &coinInventoryDBImplementation{
		db: dbConn,
	}
}

type authDBImplementation struct {
	db *sql.DB
}

func NewAuthDB(dbConn *sql.DB) AuthDB {
	return &authDBImplementation{
		db: dbConn,
	}
}

func (a *authDBImplementation) GetUserAuthData(username string) (int, string, error) {
	var (
		id           int
		passwordHash string
	)
	err := a.db.QueryRow("SELECT id, password_hash FROM users WHERE username=$1", username).
		Scan(&id, &passwordHash)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get user auth data for '%s': %w", username, err)
	}
	return id, passwordHash, nil
}

func (c *coinInventoryDBImplementation) BeginTx() (*sql.Tx, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (c *coinInventoryDBImplementation) GetCoinsForUpdate(tx *sql.Tx, userID int) (int, error) {
	var coins int
	err := tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", userID).Scan(&coins)
	if err != nil {
		return 0, fmt.Errorf("failed to get coins for user %d: %w", userID, err)
	}
	return coins, nil
}

func (c *coinInventoryDBImplementation) IncreaseCoins(tx *sql.Tx, userID int, amount int) error {
	_, err := tx.Exec("UPDATE users SET coins = coins + $1 WHERE id=$2", amount, userID)
	if err != nil {
		return fmt.Errorf("failed to increase coins: %w", err)
	}
	return nil
}

func (c *coinInventoryDBImplementation) DecreaseCoins(tx *sql.Tx, userID int, amount int) error {
	_, err := tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", amount, userID)
	if err != nil {
		return fmt.Errorf("failed to decrease coins: %w", err)
	}
	return nil
}

func (c *coinInventoryDBImplementation) IncreaseItem(tx *sql.Tx, userID int, item string, delta int) error {
	res, err := tx.Exec("UPDATE inventories SET quantity = quantity + $1 WHERE user_id=$2 AND item_type=$3", delta, userID, item)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		_, err = tx.Exec("INSERT INTO inventories (user_id, item_type, quantity) VALUES ($1, $2, $3)",
			userID, item, delta)
		if err != nil {
			return fmt.Errorf("failed to insert new item: %w", err)
		}
	}
	return nil
}

func (c *coinInventoryDBImplementation) InsertTransaction(tx *sql.Tx, userID int, transactionType, counterparty string, amount int) error {
	_, err := tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, $2, $3, $4)",
		userID, transactionType, counterparty, amount)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}
	return nil
}

func (c *coinInventoryDBImplementation) GetUserIDByUsernameForUpdate(tx *sql.Tx, username string) (int, error) {
	var userID int
	err := tx.QueryRow("SELECT id FROM users WHERE username=$1 FOR UPDATE", username).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to find user by username %q: %w", username, err)
	}
	return userID, nil
}

func (c *coinInventoryDBImplementation) InsertReceivedTransaction(tx *sql.Tx, toUserID, fromUserID, amount int) error {
	_, err := tx.Exec(`
INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) 
VALUES ($1, 'received', (SELECT username FROM users WHERE id=$2), $3)
`, toUserID, fromUserID, amount)
	if err != nil {
		return fmt.Errorf("failed to insert received transaction: %w", err)
	}
	return nil
}
func (c *coinInventoryDBImplementation) GetUserCoins(userID int) (int, error) {
	var coins int
	err := c.db.QueryRow("SELECT coins FROM users WHERE id=$1", userID).Scan(&coins)
	if err != nil {
		return 0, fmt.Errorf("failed to get user coins: %w", err)
	}
	return coins, nil
}

func (c *coinInventoryDBImplementation) GetInventory(userID int) ([]InventoryItem, error) {
	rows, err := c.db.Query("SELECT item_type, quantity FROM inventories WHERE user_id=$1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory: %w", err)
	}
	defer rows.Close()

	var items []InventoryItem
	for rows.Next() {
		var it InventoryItem
		if e2 := rows.Scan(&it.Type, &it.Quantity); e2 != nil {
			continue
		}
		items = append(items, it)
	}
	return items, nil
}

func (c *coinInventoryDBImplementation) GetTransactions(userID int, transactionType string) ([]Transaction, error) {
	rows, err := c.db.Query(`
        SELECT counterparty, amount
        FROM coin_transactions
        WHERE user_id=$1 AND transaction_type=$2
    `, userID, transactionType)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s transactions: %w", transactionType, err)
	}
	defer rows.Close()

	var trans []Transaction
	for rows.Next() {
		var t Transaction
		if e2 := rows.Scan(&t.Counterparty, &t.Amount); e2 != nil {
			continue
		}
		trans = append(trans, t)
	}
	return trans, nil
}
