package middleware

import (
	"net/http"
	"strings"

	"avito-shop/pkg"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func JWTAuthMiddleware(secret string, log pkg.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"errors": "Authorization header missing"})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			// проверка подмены токена, сделал я это для безопасности сервиса и защиты от подмены
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				log.Warn("Invalid JWT token")
				return c.JSON(http.StatusUnauthorized, map[string]string{"errors": "Invalid token"})
			}
			c.Set("user", token.Claims)
			return next(c)
		}
	}
}
