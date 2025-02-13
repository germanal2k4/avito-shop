package api

import (
	"database/sql"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthHandler(db *sql.DB, jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Invalid request"})
			return
		}

		var id int
		var passwordHash string
		query := "SELECT id, password_hash FROM users WHERE username=$1"
		err := db.QueryRow(query, req.Username).Scan(&id, &passwordHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Invalid credentials"})
			return
		}

		if req.Password != passwordHash {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Invalid credentials"})
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  id,
			"username": req.Username,
		})
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Could not generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

func InfoHandler(db *sql.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Invalid token claims"})
			return
		}
		userID := int(userClaims["user_id"].(float64))

		var coins int
		err := db.QueryRow("SELECT coins FROM users WHERE id=$1", userID).Scan(&coins)
		if err != nil {
			logger.Error("failed to query coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		rows, err := db.Query("SELECT item_type, quantity FROM inventories WHERE user_id=$1", userID)
		if err != nil {
			logger.Error("failed to query inventory", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		defer rows.Close()

		inventory := []map[string]interface{}{}
		for rows.Next() {
			var itemType string
			var quantity int
			if err := rows.Scan(&itemType, &quantity); err != nil {
				logger.Error("failed to scan inventory", zap.Error(err))
				continue
			}
			inventory = append(inventory, map[string]interface{}{
				"type":     itemType,
				"quantity": quantity,
			})
		}

		receivedRows, err := db.Query("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='received' AND user_id=$1", userID)
		if err != nil {
			logger.Error("failed to query received transactions", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		defer receivedRows.Close()

		received := []map[string]interface{}{}
		for receivedRows.Next() {
			var fromUser string
			var amount int
			if err := receivedRows.Scan(&fromUser, &amount); err != nil {
				logger.Error("failed to scan received transaction", zap.Error(err))
				continue
			}
			received = append(received, map[string]interface{}{
				"fromUser": fromUser,
				"amount":   amount,
			})
		}

		sentRows, err := db.Query("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='sent' AND user_id=$1", userID)
		if err != nil {
			logger.Error("failed to query sent transactions", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		defer sentRows.Close()

		sent := []map[string]interface{}{}
		for sentRows.Next() {
			var toUser string
			var amount int
			if err := sentRows.Scan(&toUser, &amount); err != nil {
				logger.Error("failed to scan sent transaction", zap.Error(err))
				continue
			}
			sent = append(sent, map[string]interface{}{
				"toUser": toUser,
				"amount": amount,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"coins":     coins,
			"inventory": inventory,
			"coinHistory": map[string]interface{}{
				"received": received,
				"sent":     sent,
			},
		})
	}
}

func SendCoinHandler(db *sql.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Invalid token claims"})
			return
		}
		fromUserID := int(userClaims["user_id"].(float64))

		var req struct {
			ToUser string `json:"toUser" binding:"required"`
			Amount int    `json:"amount" binding:"required,gt=0"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Invalid request body"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			logger.Error("failed to begin transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		defer tx.Rollback()

		var senderCoins int
		err = tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", fromUserID).Scan(&senderCoins)
		if err != nil {
			logger.Error("failed to get sender coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		if senderCoins < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Insufficient coins"})
			return
		}

		_, err = tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", req.Amount, fromUserID)
		if err != nil {
			logger.Error("failed to update sender coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		var toUserID int
		err = tx.QueryRow("SELECT id FROM users WHERE username=$1 FOR UPDATE", req.ToUser).Scan(&toUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Recipient not found"})
			return
		}
		_, err = tx.Exec("UPDATE users SET coins = coins + $1 WHERE id=$2", req.Amount, toUserID)
		if err != nil {
			logger.Error("failed to update recipient coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		_, err = tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, 'sent', $2, $3)", fromUserID, req.ToUser, req.Amount)
		if err != nil {
			logger.Error("failed to insert sent transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		_, err = tx.Exec("INSERT INTO coin_transactions (user_id, transaction_type, counterparty, amount) VALUES ($1, 'received', (SELECT username FROM users WHERE id=$2), $3)", toUserID, fromUserID, req.Amount)
		if err != nil {
			logger.Error("failed to insert received transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		if err := tx.Commit(); err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Coins sent successfully"})
	}
}

func BuyHandler(db *sql.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": "Invalid token claims"})
			return
		}
		userID := int(userClaims["user_id"].(float64))

		item := c.Param("item")
		if item == "" {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Item parameter is required"})
			return
		}

		cost := 100

		tx, err := db.Begin()
		if err != nil {
			logger.Error("failed to begin transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		defer tx.Rollback()

		var coins int
		err = tx.QueryRow("SELECT coins FROM users WHERE id=$1 FOR UPDATE", userID).Scan(&coins)
		if err != nil {
			logger.Error("failed to query coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}
		if coins < cost {
			c.JSON(http.StatusBadRequest, gin.H{"errors": "Insufficient coins"})
			return
		}

		_, err = tx.Exec("UPDATE users SET coins = coins - $1 WHERE id=$2", cost, userID)
		if err != nil {
			logger.Error("failed to update coins", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		var quantity int
		err = tx.QueryRow("SELECT quantity FROM inventories WHERE user_id=$1 AND item_type=$2 FOR UPDATE", userID, item).Scan(&quantity)
		if err != nil {
			_, err = tx.Exec("INSERT INTO inventories (user_id, item_type, quantity) VALUES ($1, $2, 1)", userID, item)
			if err != nil {
				logger.Error("failed to insert inventory", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
				return
			}
		} else {
			_, err = tx.Exec("UPDATE inventories SET quantity = quantity + 1 WHERE user_id=$1 AND item_type=$2", userID, item)
			if err != nil {
				logger.Error("failed to update inventory", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"errors": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Item purchased successfully"})
	}
}
