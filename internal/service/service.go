package service

import (
	"database/sql"
	"net/http"
	"sync"

	"avito-shop/internal/api"
	"avito-shop/pkg"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// апишка эх
type EchoAPI struct {
	db           *sql.DB
	jwtSecret    string
	log          pkg.Logger
	itemPricesMu sync.RWMutex
	itemPrices   map[string]int
}

// фабрика апишек
func NewEchoAPI(db *sql.DB, jwtSecret string, log pkg.Logger) *EchoAPI {
	return &EchoAPI{
		db:        db,
		jwtSecret: jwtSecret,
		log:       log,
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

// Функции для аутентификация цепляющимся к методу POST
func (a *EchoAPI) PostApiAuth(ctx echo.Context) error {
	var req api.AuthRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Invalid request")})
	}

	var id int
	var passwordHash string
	// делаем запрос в базу и достаем оттуда пароль
	err := a.db.QueryRow("SELECT id, password_hash FROM users WHERE username=$1", req.Username).Scan(&id, &passwordHash)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Invalid credentials")})
	}
	if req.Password != passwordHash {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Invalid credentials")})
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": req.Username,
	})
	// достаем токен
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Could not generate token")})
	}
	a.log.Info("User authenticated")
	return ctx.JSON(http.StatusOK, api.AuthResponse{Token: &tokenString})
}

// GET метод покупки вещей
func (a *EchoAPI) GetApiBuyItem(ctx echo.Context, item string) error {
	a.itemPricesMu.RLock()
	cost, found := a.itemPrices[item]
	a.itemPricesMu.RUnlock()
	if !found {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Item not found")})
	}
	// получаем юзера
	claims := ctx.Get("user")
	if claims == nil {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Unauthorized")})
	}
	// достаем инфу из claims
	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Invalid token claims")})
	}
	userID := int(userClaims["user_id"].(float64))

	tx, err := a.db.Begin()
	if err != nil {
		a.log.Error("failed to begin transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	defer func(tx *sql.Tx) {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			a.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}(tx)
	// достаем число монет из бд
	var coins int
	err = tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", userID).Scan(&coins)
	if err != nil {
		a.log.Error("failed to query coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	// проверяем что деньги есть
	if coins < cost {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Insufficient coins")})
	}
	_, err = tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", cost, userID)
	if err != nil {
		a.log.Error("failed to update coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	// находим количество из базы данных
	var quantity int
	err = tx.QueryRow("SELECT quantity FROM inventories WHERE user_id=$1 AND item_type=$2 FOR UPDATE", userID, item).Scan(&quantity)
	if err != nil {
		_, err = tx.Exec("INSERT INTO inventories (user_id, item_type, quantity) VALUES ($1, $2, 1)", userID, item)
		if err != nil {
			a.log.Error("failed to insert inventory", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
		}
	} else {
		_, err = tx.Exec("UPDATE inventories SET quantity = quantity + 1 WHERE user_id=$1 AND item_type=$2", userID, item)
		if err != nil {
			a.log.Error("failed to update inventory", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
		}
	}
	if err := tx.Commit(); err != nil {
		a.log.Error("failed to commit transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Item purchased successfully"})
}

// GET метод для получения информации
func (a *EchoAPI) GetApiInfo(ctx echo.Context) error {
	claims := ctx.Get("user")
	if claims == nil {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Unauthorized")})
	}
	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Invalid token claims")})
	}
	userID := int(userClaims["user_id"].(float64))

	// Получаем количество монет
	var coins int
	err := a.db.QueryRow("SELECT coins FROM users WHERE id=$1", userID).Scan(&coins)
	if err != nil {
		a.log.Error("failed to query coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}

	// Получаем инвентарь
	invRows, err := a.db.Query("SELECT item_type, quantity FROM inventories WHERE user_id=$1", userID)
	if err != nil {
		a.log.Error("failed to query inventory", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	defer invRows.Close()

	var inventoryItems []struct {
		Quantity *int    `json:"quantity,omitempty"`
		Type     *string `json:"type,omitempty"`
	}
	for invRows.Next() {
		var itemType string
		var quantity int
		if err := invRows.Scan(&itemType, &quantity); err != nil {
			a.log.Error("failed to scan inventory", zap.Error(err))
			continue
		}
		t := itemType
		q := quantity
		inventoryItems = append(inventoryItems, struct {
			Quantity *int    `json:"quantity,omitempty"`
			Type     *string `json:"type,omitempty"`
		}{
			Quantity: &q,
			Type:     &t,
		})
	}

	// Получаем историю транзакций (полученные монеты)
	recRows, err := a.db.Query("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='received' AND user_id=$1", userID)
	if err != nil {
		a.log.Error("failed to query received transactions", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	defer recRows.Close()

	var receivedTrans []struct {
		Amount   *int    `json:"amount,omitempty"`
		FromUser *string `json:"fromUser,omitempty"`
	}
	for recRows.Next() {
		var fromUser string
		var amount int
		if err := recRows.Scan(&fromUser, &amount); err != nil {
			a.log.Error("failed to scan received transaction", zap.Error(err))
			continue
		}
		fu := fromUser
		amt := amount
		receivedTrans = append(receivedTrans, struct {
			Amount   *int    `json:"amount,omitempty"`
			FromUser *string `json:"fromUser,omitempty"`
		}{
			Amount:   &amt,
			FromUser: &fu,
		})
	}

	sentRows, err := a.db.Query("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='sent' AND user_id=$1", userID)
	if err != nil {
		a.log.Error("failed to query sent transactions", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	defer sentRows.Close()

	var sentTrans []struct {
		Amount *int    `json:"amount,omitempty"`
		ToUser *string `json:"toUser,omitempty"`
	}
	for sentRows.Next() {
		var toUser string
		var amount int
		if err := sentRows.Scan(&toUser, &amount); err != nil {
			a.log.Error("failed to scan sent transaction", zap.Error(err))
			continue
		}
		tu := toUser
		amt := amount
		sentTrans = append(sentTrans, struct {
			Amount *int    `json:"amount,omitempty"`
			ToUser *string `json:"toUser,omitempty"`
		}{
			Amount: &amt,
			ToUser: &tu,
		})
	}

	info := api.InfoResponse{
		Coins:     &coins,
		Inventory: &inventoryItems,
		CoinHistory: &struct {
			Received *[]struct {
				Amount   *int    `json:"amount,omitempty"`
				FromUser *string `json:"fromUser,omitempty"`
			} `json:"received,omitempty"`
			Sent *[]struct {
				Amount *int    `json:"amount,omitempty"`
				ToUser *string `json:"toUser,omitempty"`
			} `json:"sent,omitempty"`
		}{
			Received: &receivedTrans,
			Sent:     &sentTrans,
		},
	}
	return ctx.JSON(http.StatusOK, info)
}

func (a *EchoAPI) PostApiSendCoin(ctx echo.Context) error {
	claims := ctx.Get("user")
	if claims == nil {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Unauthorized")})
	}
	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, api.ErrorResponse{Errors: ptr("Invalid token claims")})
	}
	fromUserID := int(userClaims["user_id"].(float64))
	var req api.SendCoinRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Invalid request body")})
	}
	tx, err := a.db.Begin()
	if err != nil {
		a.log.Error("failed to begin transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	defer func(tx *sql.Tx) {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			a.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}(tx)
	var senderCoins int
	err = tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", fromUserID).Scan(&senderCoins)
	if err != nil {
		a.log.Error("failed to get sender coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	if senderCoins < req.Amount {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Insufficient coins")})
	}
	_, err = tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", req.Amount, fromUserID)
	if err != nil {
		a.log.Error("failed to update sender coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	var toUserID int
	err = tx.QueryRow("SELECT id FROM users WHERE username=$1 FOR UPDATE", req.ToUser).Scan(&toUserID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Errors: ptr("Recipient not found")})
	}
	_, err = tx.Exec("UPDATE users SET coins = coins + $1 WHERE id=$2", req.Amount, toUserID)
	if err != nil {
		a.log.Error("failed to update recipient coins", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	_, err = tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, 'sent', $2, $3)", fromUserID, req.ToUser, req.Amount)
	if err != nil {
		a.log.Error("failed to insert sent transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	_, err = tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, 'received', (SELECT username FROM users WHERE id=$2), $3)", toUserID, fromUserID, req.Amount)
	if err != nil {
		a.log.Error("failed to insert received transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	if err := tx.Commit(); err != nil {
		a.log.Error("failed to commit transaction", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Errors: ptr("Internal server error")})
	}
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Coins sent successfully"})
}

func ptr(s string) *string {
	return &s
}
