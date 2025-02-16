package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type mockAuthDB struct {
	GetUserAuthDataFunc func(username string) (int, string, error)
}

func (m *mockAuthDB) GetUserAuthData(username string) (int, string, error) {
	return m.GetUserAuthDataFunc(username)
}
func TestAuthService_Authenticate_Success(t *testing.T) {
	mockDB := &mockAuthDB{
		GetUserAuthDataFunc: func(username string) (int, string, error) {
			if username == "testuser" {
				return 1, "secret", nil
			}
			return 0, "", errors.New("not found")
		},
	}
	logger := &mockLogger{}
	authSvc := NewAuthService(mockDB, logger, "jwtSecret")

	tokenStr, err := authSvc.Authenticate("testuser", "secret")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tokenStr == "" {
		t.Errorf("expected non-empty token")
	}
	parsed, err := jwt.Parse(tokenStr, func(tok *jwt.Token) (interface{}, error) {
		return []byte("jwtSecret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Errorf("failed to parse or invalid token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Errorf("unexpected claims type: %T", parsed.Claims)
	}
	if claims["user_id"] != float64(1) || claims["username"] != "testuser" {
		t.Errorf("claims mismatch: %v", claims)
	}
	if exp, ok := claims["exp"].(float64); !ok || exp < float64(time.Now().Unix()) {
		t.Errorf("token exp is not set properly: %v", claims["exp"])
	}
}

func TestAuthService_Authenticate_UserNotFound(t *testing.T) {
	mockDB := &mockAuthDB{
		GetUserAuthDataFunc: func(username string) (int, string, error) {
			return 0, "", errors.New("no rows")
		},
	}
	authSvc := NewAuthService(mockDB, &mockLogger{}, "secretJWT")

	tokenStr, err := authSvc.Authenticate("unknownUser", "anyPass")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected 'invalid credentials', got: %v", err)
	}
	if tokenStr != "" {
		t.Errorf("expected empty token, got: %s", tokenStr)
	}
}

func TestAuthService_Authenticate_WrongPassword(t *testing.T) {
	mockDB := &mockAuthDB{
		GetUserAuthDataFunc: func(username string) (int, string, error) {
			return 2, "realPass", nil
		},
	}
	authSvc := NewAuthService(mockDB, &mockLogger{}, "someSecret")

	tokenStr, err := authSvc.Authenticate("someuser", "wrongPass")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected 'invalid credentials', got: %v", err)
	}
	if tokenStr != "" {
		t.Errorf("expected empty token when error, got: %s", tokenStr)
	}
}

func TestAuthService_Authenticate_JWTError(t *testing.T) {
	mockDB := &mockAuthDB{
		GetUserAuthDataFunc: func(username string) (int, string, error) {
			return 1, "secretPass", nil
		},
	}
	logger := &mockLogger{}

	authSvc := NewAuthService(mockDB, logger, "")

	tokenStr, err := authSvc.Authenticate("testuser", "secretPass")
	if err == nil {
		t.Fatalf("expected error generating token, got nil")
	}
	if !strings.Contains(err.Error(), "could not generate token: empty secret") {
		t.Errorf("unexpected error: %v", err)
	}
	if tokenStr != "" {
		t.Errorf("expected empty token, got %s", tokenStr)
	}
}
