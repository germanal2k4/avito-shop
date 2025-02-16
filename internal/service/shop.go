package service

import (
	"avito-shop/internal/db"
	"avito-shop/pkg"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

var (
	ErrNotEnoughCoins = errors.New("not enough coins")
	ErrItemNotFound   = errors.New("item not found")
	ErrUserNotFound   = errors.New("user not found")
)

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
	FromUser string
	ToUser   string
	Amount   int
}

type ShopService interface {
	BuyItem(userID int, item string) error

	SendCoins(fromUserID int, toUsername string, amount int) error

	GetCoins(userID int) (int, error)

	GetUserInfo(userID int) (Info, error)
}

type shopService struct {
	dbProv     db.CoinInventoryDB
	log        pkg.Logger
	itemPrices map[string]int
}

func NewShopService(dbProv db.CoinInventoryDB, log pkg.Logger) ShopService {
	return &shopService{
		dbProv: dbProv,
		log:    log,
		itemPrices: map[string]int{
			"t-shirt":    80,
			"cup":        20,
			"book":       50,
			"pen":        10,
			"powerbank":  200,
			"hoody":      300,
			"umbrella":   200,
			"socks":      10,
			"wallet":     50,
			"pink-hoody": 500,
		},
	}
}

func (s *shopService) BuyItem(userID int, item string) error {
	tx, err := s.dbProv.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	coins, err := s.dbProv.GetCoinsForUpdate(tx, userID)
	if err != nil {
		s.log.Error("failed to get user coins for update", zap.Int("userID", userID), zap.Error(err))
		return err
	}
	cost, ok := s.itemPrices[item]
	if !ok {
		return ErrItemNotFound
	}
	if coins < cost {
		return ErrNotEnoughCoins
	}

	if err := s.dbProv.DecreaseCoins(tx, userID, cost); err != nil {
		s.log.Error("failed to decrease user coins", zap.Int("userID", userID), zap.Error(err))
		return err
	}

	if err := s.dbProv.IncreaseItem(tx, userID, item, 1); err != nil {
		s.log.Error("failed to increase item", zap.Int("userID", userID), zap.String("item", item), zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("failed to commit buy item", zap.Int("userID", userID), zap.String("item", item), zap.Error(err))
		return err
	}
	s.log.Info("Item purchased successfully", zap.Int("userID", userID), zap.String("item", item))
	return nil
}

func (s *shopService) SendCoins(fromUserID int, toUsername string, amount int) error {
	tx, err := s.dbProv.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	senderCoins, err := s.dbProv.GetCoinsForUpdate(tx, fromUserID)
	if err != nil {
		s.log.Error("failed to get sender coins", zap.Int("fromUserID", fromUserID), zap.Error(err))
		return err
	}
	if senderCoins < amount {
		return ErrNotEnoughCoins
	}

	if err := s.dbProv.DecreaseCoins(tx, fromUserID, amount); err != nil {
		s.log.Error("failed to decrease sender coins", zap.Int("fromUserID", fromUserID), zap.Error(err))
		return err
	}

	toUserID, err := s.dbProv.GetUserIDByUsernameForUpdate(tx, toUsername)
	if err != nil {
		s.log.Warn("recipient not found", zap.String("toUsername", toUsername), zap.Error(err))
		return ErrUserNotFound
	}

	if err := s.dbProv.IncreaseCoins(tx, toUserID, amount); err != nil {
		s.log.Error("failed to increase recipient coins", zap.Int("toUserID", toUserID), zap.Error(err))
		return err
	}

	if err := s.dbProv.InsertTransaction(tx, fromUserID, "sent", toUsername, amount); err != nil {
		s.log.Error("failed to insert sent transaction", zap.Error(err))
		return err
	}

	if err := s.dbProv.InsertReceivedTransaction(tx, toUserID, fromUserID, amount); err != nil {
		s.log.Error("failed to insert received transaction", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("failed to commit send coins", zap.Error(err))
		return err
	}
	s.log.Info("Coins sent successfully",
		zap.Int("fromUserID", fromUserID),
		zap.String("toUsername", toUsername),
		zap.Int("amount", amount))
	return nil
}

func (s *shopService) GetCoins(userID int) (int, error) {
	coins, err := s.dbProv.GetUserCoins(userID)
	if err != nil {
		s.log.Error("failed to get user coins", zap.Int("userID", userID), zap.Error(err))
		return 0, err
	}
	return coins, nil
}

func (s *shopService) GetUserInfo(userID int) (Info, error) {
	var info Info

	coins, err := s.dbProv.GetUserCoins(userID)
	if err != nil {
		s.log.Error("failed to get user coins", zap.Int("userID", userID), zap.Error(err))
		return Info{}, err
	}
	info.Coins = coins

	invItems, err := s.dbProv.GetInventory(userID)
	if err != nil {
		s.log.Error("failed to get inventory", zap.Int("userID", userID), zap.Error(err))
		return Info{}, err
	}
	var inventory []InventoryItem
	for _, it := range invItems {
		inventory = append(inventory, InventoryItem{
			Type:     it.Type,
			Quantity: it.Quantity,
		})
	}
	info.Inventory = inventory

	receivedDB, err := s.dbProv.GetTransactions(userID, "received")
	if err != nil {
		s.log.Error("failed to get received transactions", zap.Int("userID", userID), zap.Error(err))
		return Info{}, err
	}
	var received []Transaction
	for _, r := range receivedDB {
		received = append(received, Transaction{
			FromUser: r.Counterparty,
			Amount:   r.Amount,
		})
	}

	sentDB, err := s.dbProv.GetTransactions(userID, "sent")
	if err != nil {
		s.log.Error("failed to get sent transactions", zap.Int("userID", userID), zap.Error(err))
		return Info{}, err
	}
	var sent []Transaction
	for _, sTr := range sentDB {
		sent = append(sent, Transaction{
			ToUser: sTr.Counterparty,
			Amount: sTr.Amount,
		})
	}
	info.CoinHistory = CoinHistory{
		Received: received,
		Sent:     sent,
	}

	return info, nil
}
