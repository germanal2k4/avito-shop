package service

import (
	"avito-shop/internal/db"
	"avito-shop/pkg"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type AuthService interface {
	Authenticate(username, password string) (string, error)
}

type authService struct {
	authDB    db.AuthDB
	log       pkg.Logger
	jwtSecret string
}

func NewAuthService(authDB db.AuthDB, logger pkg.Logger, jwtSecret string) AuthService {
	return &authService{
		authDB:    authDB,
		log:       logger,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) Authenticate(username, password string) (string, error) {
	if s.jwtSecret == "" {
		s.log.Error("auth: empty JWT secret key")
		return "", errors.New("could not generate token: empty secret key")
	}
	id, passHash, err := s.authDB.GetUserAuthData(username)
	if err != nil {
		s.log.Warn("invalid credentials", zap.String("username", username), zap.Error(err))
		return "", fmt.Errorf("invalid credentials: %w", err)
	}
	if passHash != password {
		s.log.Warn("invalid credentials: password mismatch", zap.String("username", username))
		return "", fmt.Errorf("invalid credentials: password mismatch")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": username,
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.log.Error("failed to generate token", zap.String("username", username), zap.Error(err))
		return "", fmt.Errorf("could not generate token: %w", err)
	}
	s.log.Info("User authenticated", zap.Int("userID", id), zap.String("username", username))
	return tokenString, nil
}
