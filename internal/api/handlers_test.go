package api

import (
	"avito-shop/internal/config"
	"avito-shop/internal/db"
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// setupRouter создаёт Gin-роутер для тестирования
func setupRouter(db *sql.DB, jwtSecret string, logger *zap.Logger) *gin.Engine {
	router := gin.New()
	router.POST("/api/auth", AuthHandler(db, jwtSecret, logger))
	return router
}

func TestAuthHandler_InvalidBody(t *testing.T) {
	logger := zap.NewNop()
	router := setupRouter(nil, "secret", logger)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth", bytes.NewBufferString(`{invalid json}`))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_ValidCredentials(t *testing.T) {
	logger := zap.NewNop()

	cfg := &config.Config{
		DatabaseHost:     os.Getenv("DATABASE_HOST"),
		DatabasePort:     os.Getenv("DATABASE_PORT"),
		DatabaseUser:     os.Getenv("DATABASE_USER"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:     os.Getenv("DATABASE_NAME"),
	}
	dbConn, err := db.Connect(cfg)
	if err != nil {
		t.Fatal("failed to connect to database", err)
	}
	defer dbConn.Close()

	// Подготавливаем тестового пользователя
	_, err = dbConn.Exec("INSERT INTO users (username, password_hash, coins) VALUES ('testuser', 'testpass', 1000) ON CONFLICT (username) DO NOTHING")
	if err != nil {
		t.Fatal("failed to insert test user", err)
	}

	router := setupRouter(dbConn, "secret", logger)

	body := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
