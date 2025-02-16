package models

type Info struct {
	Coins       int
	Inventory   []InventoryItem
	CoinHistory CoinHistory
}

type InventoryItem struct {
	Type     string
	Quantity int
}

type CoinHistory struct {
	Received []Transaction
	Sent     []Transaction
}

type Transaction struct {
	Counterparty string
	Amount       int
}
