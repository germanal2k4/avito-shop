package service

import (
	"avito-shop/internal/db"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Error(msg string, fields ...zap.Field) {}
func (m *mockLogger) Sync() error                           { return nil }

type mockCoinDB struct {
	GetUserCoinsFunc    func(int) (int, error)
	GetInventoryFunc    func(int) ([]db.InventoryItem, error)
	GetTransactionsFunc func(int, string) ([]db.Transaction, error)
}

func (m *mockCoinDB) BeginTx() (*sql.Tx, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) GetCoinsForUpdate(tx *sql.Tx, userID int) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) IncreaseCoins(tx *sql.Tx, userID, amount int) error {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) DecreaseCoins(tx *sql.Tx, userID, amount int) error {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) IncreaseItem(tx *sql.Tx, userID int, item string, delta int) error {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) InsertTransaction(tx *sql.Tx, userID int, transactionType, counterparty string, amount int) error {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) InsertReceivedTransaction(tx *sql.Tx, toUserID, fromUserID, amount int) error {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) GetUserIDByUsernameForUpdate(tx *sql.Tx, username string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCoinDB) GetUserCoins(userID int) (int, error) {
	return m.GetUserCoinsFunc(userID)
}

func (m *mockCoinDB) GetInventory(userID int) ([]db.InventoryItem, error) {
	return m.GetInventoryFunc(userID)
}

func (m *mockCoinDB) GetTransactions(userID int, ttype string) ([]db.Transaction, error) {
	return m.GetTransactionsFunc(userID, ttype)
}

type coinInventorySQLMock struct {
	db *sql.DB
}

func (c *coinInventorySQLMock) BeginTx() (*sql.Tx, error) {
	return c.db.Begin()
}

func (c *coinInventorySQLMock) GetCoinsForUpdate(tx *sql.Tx, userID int) (int, error) {
	var coins int
	err := tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", userID).Scan(&coins)
	return coins, err
}

func (c *coinInventorySQLMock) IncreaseCoins(tx *sql.Tx, userID, amount int) error {
	_, err := tx.Exec("UPDATE users SET coins = coins + $1 WHERE id=$2", amount, userID)
	return err
}

func (c *coinInventorySQLMock) DecreaseCoins(tx *sql.Tx, userID, amount int) error {
	_, err := tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", amount, userID)
	return err
}

func (c *coinInventorySQLMock) IncreaseItem(tx *sql.Tx, userID int, item string, delta int) error {
	row := tx.QueryRow("SELECT quantity FROM inventories WHERE user_id=$1 AND item_type=$2 FOR UPDATE", userID, item)
	var q int
	err := row.Scan(&q)
	if err == sql.ErrNoRows {
		_, e2 := tx.Exec("INSERT INTO inventories (user_id, item_type, quantity) VALUES ($1, $2, $3)",
			userID, item, delta)
		return e2
	} else if err != nil {
		return err
	}
	_, e3 := tx.Exec("UPDATE inventories SET quantity = quantity + $1 WHERE user_id=$2 AND item_type=$3",
		delta, userID, item)
	return e3
}

func (c *coinInventorySQLMock) InsertTransaction(tx *sql.Tx, userID int, transactionType, counterparty string, amount int) error {
	_, err := tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, $2, $3, $4)",
		userID, transactionType, counterparty, amount)
	return err
}

func (c *coinInventorySQLMock) InsertReceivedTransaction(tx *sql.Tx, toUserID, fromUserID, amount int) error {
	_, err := tx.Exec(`
    INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount)
    VALUES ($1, 'received', (SELECT username FROM users WHERE id=$2), $3)`,
		toUserID, fromUserID, amount)
	return err
}

func (c *coinInventorySQLMock) GetUserIDByUsernameForUpdate(tx *sql.Tx, username string) (int, error) {
	var uid int
	err := tx.QueryRow("SELECT id FROM users WHERE username=$1 FOR UPDATE", username).Scan(&uid)
	return uid, err
}

func (c *coinInventorySQLMock) GetUserCoins(userID int) (int, error) {
	var coins int
	err := c.db.QueryRow("SELECT coins FROM users WHERE id=$1", userID).Scan(&coins)
	return coins, err
}

func (c *coinInventorySQLMock) GetInventory(userID int) ([]db.InventoryItem, error) {
	rows, err := c.db.Query("SELECT item_type, quantity FROM inventories WHERE user_id=$1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []db.InventoryItem
	for rows.Next() {
		var it db.InventoryItem
		if e2 := rows.Scan(&it.Type, &it.Quantity); e2 == nil {
			items = append(items, it)
		}
	}
	return items, nil
}

func (c *coinInventorySQLMock) GetTransactions(userID int, ttype string) ([]db.Transaction, error) {
	rows, err := c.db.Query("SELECT counterparty, amount FROM coin_transactions WHERE user_id=$1 AND transaction_type=$2",
		userID, ttype)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trans []db.Transaction
	for rows.Next() {
		var t db.Transaction
		if e2 := rows.Scan(&t.Counterparty, &t.Amount); e2 == nil {
			trans = append(trans, t)
		}
	}
	return trans, nil
}

func TestShopService_BuyItem_Success(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbConn.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(100))

	mock.ExpectExec("UPDATE users SET coins = coins - \\$1 WHERE id=\\$2").
		WithArgs(20, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT quantity FROM inventories WHERE user_id=\\$1 AND item_type=\\$2 FOR UPDATE").
		WithArgs(1, "cup").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO inventories").
		WithArgs(1, "cup", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	svc := &shopService{
		dbProv: &coinInventorySQLMock{db: dbConn},
		log:    &mockLogger{},
		itemPrices: map[string]int{
			"cup": 20,
		},
	}

	if err := svc.BuyItem(1, "cup"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestShopService_BuyItem_NotEnoughCoins(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer dbConn.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(10))
	mock.ExpectRollback()

	svc := &shopService{
		dbProv: &coinInventorySQLMock{db: dbConn},
		log:    &mockLogger{},
		itemPrices: map[string]int{
			"cup": 20,
		},
	}

	err = svc.BuyItem(1, "cup")
	if !errors.Is(err, ErrNotEnoughCoins) {
		t.Errorf("expected ErrNotEnoughCoins, got %v", err)
	}
	if e2 := mock.ExpectationsWereMet(); e2 != nil {
		t.Errorf("unmet expecations: %v", e2)
	}
}

func TestShopService_BuyItem_ItemNotFound(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error: %v", err)
	}
	defer dbConn.Close()

	mock.ExpectBegin()

	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(100))

	mock.ExpectRollback()

	svc := &shopService{
		dbProv:     &coinInventorySQLMock{db: dbConn},
		log:        &mockLogger{},
		itemPrices: map[string]int{},
	}

	err = svc.BuyItem(1, "cup")
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}

	if e2 := mock.ExpectationsWereMet(); e2 != nil {
		t.Errorf("unmet expectations: %v", e2)
	}
}

func TestShopService_SendCoins_Success(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer dbConn.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(100))
	mock.ExpectExec("UPDATE users SET coins = coins - \\$1 WHERE id=\\$2").
		WithArgs(30, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM users WHERE username=\\$1 FOR UPDATE").
		WithArgs("otheruser").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectExec("UPDATE users SET coins = coins \\+ \\$1 WHERE id=\\$2").
		WithArgs(30, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO coin_transactions").
		WithArgs(1, "sent", "otheruser", 30).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO coin_transactions").
		WithArgs(2, 1, 30).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	svc := &shopService{
		dbProv: &coinInventorySQLMock{db: dbConn},
		log:    &mockLogger{},
		itemPrices: map[string]int{
			"cup": 20,
		},
	}

	err = svc.SendCoins(1, "otheruser", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e2 := mock.ExpectationsWereMet(); e2 != nil {
		t.Errorf("unmet: %v", e2)
	}
}
func TestShopService_SendCoins_NotEnoughCoins(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer dbConn.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(20))
	mock.ExpectRollback()

	svc := &shopService{
		dbProv: &coinInventorySQLMock{db: dbConn},
		log:    &mockLogger{},
	}

	err = svc.SendCoins(1, "otheruser", 30)
	if !errors.Is(err, ErrNotEnoughCoins) {
		t.Errorf("expected ErrNotEnoughCoins, got %v", err)
	}

	if e2 := mock.ExpectationsWereMet(); e2 != nil {
		t.Errorf("unmet expectations: %v", e2)
	}
}

func TestShopService_GetUserInfo_Success(t *testing.T) {
	mockDB := &mockCoinDB{
		GetUserCoinsFunc: func(userID int) (int, error) {
			return 300, nil
		},
		GetInventoryFunc: func(userID int) ([]db.InventoryItem, error) {
			return []db.InventoryItem{
				{Type: "cup", Quantity: 2},
			}, nil
		},
		GetTransactionsFunc: func(userID int, ttype string) ([]db.Transaction, error) {
			if ttype == "received" {
				return []db.Transaction{
					{Counterparty: "Alice", Amount: 50},
				}, nil
			}
			return []db.Transaction{
				{Counterparty: "Bob", Amount: 20},
			}, nil
		},
	}
	logger := &mockLogger{}

	svc := &shopService{
		dbProv: mockDB,
		log:    logger,
	}

	info, err := svc.GetUserInfo(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Coins != 300 {
		t.Errorf("expected 300 coins, got %d", info.Coins)
	}
	if len(info.Inventory) != 1 || info.Inventory[0].Type != "cup" || info.Inventory[0].Quantity != 2 {
		t.Errorf("unexpected inventory: %v", info.Inventory)
	}
	if len(info.CoinHistory.Received) != 1 {
		t.Errorf("expected 1 received, got %d", len(info.CoinHistory.Received))
	} else {
		if info.CoinHistory.Received[0].FromUser != "Alice" || info.CoinHistory.Received[0].Amount != 50 {
			t.Errorf("received mismatch: %v", info.CoinHistory.Received)
		}
	}
	if len(info.CoinHistory.Sent) != 1 {
		t.Errorf("expected 1 sent, got %d", len(info.CoinHistory.Sent))
	} else {
		if info.CoinHistory.Sent[0].ToUser != "Bob" || info.CoinHistory.Sent[0].Amount != 20 {
			t.Errorf("sent mismatch: %v", info.CoinHistory.Sent)
		}
	}
}
