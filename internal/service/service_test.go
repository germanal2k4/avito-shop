package service

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"avito-shop/internal/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func createEchoContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestBuyItem_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error initializing sqlmock: %v", err)
	}
	defer db.Close()

	// Используем существующий товар из мапы (например, "cup" стоимостью 20)
	// Ожидаем успешное выполнение транзакции:
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1 FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(100))
	mock.ExpectExec("UPDATE users SET coins = coins - \\$1 WHERE id=\\$2").
		WithArgs(20, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Для инвентаря: если записи нет, вставка
	mock.ExpectQuery("SELECT quantity FROM inventories WHERE user_id=\\$1 AND item_type=\\$2 FOR UPDATE").
		WithArgs(1, "cup").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO inventories").
		WithArgs(1, "cup").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	echoAPI := NewEchoAPI(db, "secret", zap.NewNop())
	ctx, rec := createEchoContext(http.MethodGet, "/api/buy/cup", "")
	claims := jwt.MapClaims{"user_id": float64(1)}
	ctx.Set("user", claims)

	err = echoAPI.GetApiBuyItem(ctx, "cup")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	// Проверяем, что ответ содержит сообщение об успешной покупке
	if !strings.Contains(rec.Body.String(), "Item purchased successfully") {
		t.Errorf("Unexpected response body: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestPostApiSendCoin_Success проверяет успешный перевод монет.
func TestPostApiSendCoin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error initializing sqlmock: %v", err)
	}
	defer db.Close()

	// Смоделируем успешный перевод:
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT coins FROM users WHERE id=\$1 FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(150))
	mock.ExpectExec(`UPDATE users SET coins = coins - \$1 WHERE id=\$2`).
		WithArgs(50, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id FROM users WHERE username=\$1 FOR UPDATE`).
		WithArgs("otheruser").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectExec(`UPDATE users SET coins = coins \+ \$1 WHERE id=\$2`).
		WithArgs(50, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO coin_transactions \(user_id, transaction_type, counterparty, amount\) VALUES \(\$1, 'sent', \$2, \$3\)`).
		WithArgs(1, "otheruser", 50).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO coin_transactions \(user_id, transaction_type, counterparty, amount\) VALUES \(\$1, 'received', \(SELECT username FROM users WHERE id=\$2\), \$3\)`).
		WithArgs(2, 1, 50).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	echoAPI := NewEchoAPI(db, "secret", zap.NewNop())
	ctx, rec := createEchoContext(http.MethodPost, "/api/sendCoin", `{"toUser":"otheruser","amount":50}`)
	claims := jwt.MapClaims{"user_id": float64(1)}
	ctx.Set("user", claims)

	err = echoAPI.PostApiSendCoin(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Coins sent successfully") {
		t.Errorf("Unexpected response body: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestPostApiAuth_InvalidCredentials проверяет сценарий неверных учетных данных.
func TestPostApiAuth_InvalidCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error initializing sqlmock: %v", err)
	}
	defer db.Close()

	// Ожидаем, что пользователь не найден или пароль не совпадает.
	mock.ExpectQuery("SELECT id, password_hash FROM users WHERE username=\\$1").
		WithArgs("baduser").
		WillReturnError(sql.ErrNoRows)

	echoAPI := NewEchoAPI(db, "secret", zap.NewNop())
	ctx, rec := createEchoContext(http.MethodPost, "/api/auth", `{"username":"baduser","password":"wrong"}`)

	err = echoAPI.PostApiAuth(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid credentials") {
		t.Errorf("Unexpected response body: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetApiInfo_WithData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error initializing sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT coins FROM users WHERE id=\\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(250))

	mock.ExpectQuery("SELECT item_type, quantity FROM inventories WHERE user_id=\\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "quantity"}))

	mock.ExpectQuery("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='received' AND user_id=\\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"counterparty", "amount"}))

	mock.ExpectQuery("SELECT counterparty, amount FROM coin_transactions WHERE transaction_type='sent' AND user_id=\\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"counterparty", "amount"}))

	echoAPI := NewEchoAPI(db, "secret", zap.NewNop())
	ctx, rec := createEchoContext(http.MethodGet, "/api/info", "")
	claims := jwt.MapClaims{"user_id": float64(1)}
	ctx.Set("user", claims)

	err = echoAPI.GetApiInfo(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var infoResp api.InfoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &infoResp); err != nil {
		t.Fatalf("Error unmarshaling response: %v", err)
	}
	if infoResp.Coins == nil || *infoResp.Coins != 250 {
		t.Errorf("Expected coins to be 250, got %v", infoResp.Coins)
	}
	if infoResp.Inventory != nil && len(*infoResp.Inventory) != 0 {
		t.Errorf("Expected empty inventory, got %v", infoResp.Inventory)
	}
	if infoResp.CoinHistory != nil {
		if infoResp.CoinHistory.Received != nil && len(*infoResp.CoinHistory.Received) != 0 {
			t.Errorf("Expected empty received transactions, got %v", infoResp.CoinHistory.Received)
		}
		if infoResp.CoinHistory.Sent != nil && len(*infoResp.CoinHistory.Sent) != 0 {
			t.Errorf("Expected empty sent transactions, got %v", infoResp.CoinHistory.Sent)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
