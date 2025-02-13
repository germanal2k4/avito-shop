package models

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Coins        int
}

type Inventory struct {
	ID       int
	UserID   int
	ItemType string
	Quantity int
}

type CoinTransaction struct {
	ID              int
	UserID          int
	TransactionType string
	Counterparty    string
	Amount          int
}
