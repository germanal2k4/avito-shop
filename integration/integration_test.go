package integration

import (
	"avito-shop/internal/api"
	"avito-shop/internal/config"
	"avito-shop/internal/db"
	"avito-shop/internal/middleware"
	"avito-shop/internal/service"
	"avito-shop/pkg"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func setupTestDB(t *testing.T) *sql.DB {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	dbConn, err := db.Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	db.Migrate(dbConn, "../migrations")
	_, err = dbConn.Exec("TRUNCATE TABLE coin_transactions, inventories, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
	return dbConn
}

func createTestServer(dbConn *sql.DB, cfg *config.Config, log pkg.Logger) *echo.Echo {
	e := echo.New()
	e.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret, log))
	apiImpl := service.NewEchoAPI(dbConn, cfg.JWTSecret, log)
	api.RegisterHandlers(e, apiImpl)
	return e
}

func registerTestUser(dbConn *sql.DB, username, password string, coins int) (int, error) {
	var id int
	err := dbConn.QueryRow(
		"INSERT INTO users (username, password_hash, coins) VALUES ($1, $2, $3) RETURNING id",
		username, password, coins,
	).Scan(&id)
	return id, err
}

func generateToken(jwtSecret string, userID int, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(jwtSecret))
}

func TestIntegration_BuyMerch(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	dbConn := setupTestDB(t)
	defer dbConn.Close()

	userID, err := registerTestUser(dbConn, "buyer", "pass", 500)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	token, err := generateToken(cfg.JWTSecret, userID, "buyer")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	e := createTestServer(dbConn, cfg, zap.NewNop())
	ts := httptest.NewServer(e)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/buy/t-shirt", ts.URL), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if msg, ok := body["message"]; !ok || msg != "Item purchased successfully" {
		t.Errorf("unexpected response message: %v", body)
	}
}

func TestIntegration_SendCoins(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	dbConn := setupTestDB(t)
	defer dbConn.Close()

	senderID, err := registerTestUser(dbConn, "sender", "pass", 500)
	if err != nil {
		t.Fatalf("failed to register sender: %v", err)
	}
	_, err = registerTestUser(dbConn, "recipient", "pass", 100)
	if err != nil {
		t.Fatalf("failed to register recipient: %v", err)
	}

	token, err := generateToken(cfg.JWTSecret, senderID, "sender")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	e := createTestServer(dbConn, cfg, zap.NewNop())
	ts := httptest.NewServer(e)
	defer ts.Close()

	reqBody := `{"toUser":"recipient","amount":50}`
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/sendCoin", ts.URL), strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if msg, ok := body["message"]; !ok || msg != "Coins sent successfully" {
		t.Errorf("unexpected response message: %v", body)
	}
}
