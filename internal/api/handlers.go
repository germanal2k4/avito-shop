package api

import (
	"avito-shop/internal/service"
	"avito-shop/pkg"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

type Handlers struct {
	AuthService service.AuthService
	ShopService service.ShopService
	Logger      pkg.Logger
	JWTSecret   string
}

var _ ServerInterface = (*Handlers)(nil)

func (h *Handlers) PostApiAuth(ctx echo.Context) error {
	var req AuthRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Invalid request body")})
	}

	token, err := h.AuthService.Authenticate(req.Username, req.Password)
	if err != nil {
		h.Logger.Warn("invalid credentials", zap.String("username", req.Username), zap.Error(err))
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Errors: ptr("Invalid credentials")})
	}
	return ctx.JSON(http.StatusOK, AuthResponse{Token: &token})
}

func (h *Handlers) GetApiBuyItem(ctx echo.Context, item string) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Errors: ptr(err.Error())})
	}

	err = h.ShopService.BuyItem(userID, item)
	if err != nil {
		if errors.Is(err, service.ErrNotEnoughCoins) {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Not enough coins")})
		}
		if errors.Is(err, service.ErrItemNotFound) {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Item not found")})
		}
		h.Logger.Error("failed to buy item", zap.Int("userID", userID), zap.String("item", item), zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Errors: ptr("Internal server error")})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Item purchased successfully"})
}

func (h *Handlers) GetApiInfo(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Errors: ptr(err.Error())})
	}

	info, err := h.ShopService.GetUserInfo(userID)
	if err != nil {
		h.Logger.Error("failed to get user info", zap.Int("userID", userID), zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Errors: ptr("Internal server error")})
	}

	return ctx.JSON(http.StatusOK, convertToInfoResponse(info))
}

func (h *Handlers) PostApiSendCoin(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Errors: ptr(err.Error())})
	}

	var req SendCoinRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Invalid request body")})
	}
	if req.Amount <= 0 {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Amount must be > 0")})
	}

	err = h.ShopService.SendCoins(userID, req.ToUser, req.Amount)
	if err != nil {
		if errors.Is(err, service.ErrNotEnoughCoins) {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Not enough coins")})
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Errors: ptr("Recipient not found")})
		}
		h.Logger.Error("failed to send coins", zap.Int("fromUserID", userID), zap.String("toUser", req.ToUser), zap.Int("amount", req.Amount), zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Errors: ptr("Internal server error")})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Coins sent successfully"})
}

func getUserIDFromContext(ctx echo.Context) (int, error) {
	claims := ctx.Get("user")
	if claims == nil {
		return 0, errUnauthorized("Unauthorized")
	}
	jwtClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return 0, errUnauthorized("Invalid token claims")
	}
	uidFloat, ok := jwtClaims["user_id"].(float64)
	if !ok {
		return 0, errUnauthorized("Invalid token claims")
	}
	return int(uidFloat), nil
}

func convertToInfoResponse(info service.Info) InfoResponse {
	var inv []struct {
		Quantity *int    `json:"quantity,omitempty"`
		Type     *string `json:"type,omitempty"`
	}
	for _, it := range info.Inventory {
		q := it.Quantity
		t := it.Type
		inv = append(inv, struct {
			Quantity *int    `json:"quantity,omitempty"`
			Type     *string `json:"type,omitempty"`
		}{
			Quantity: &q,
			Type:     &t,
		})
	}

	var received []struct {
		Amount   *int    `json:"amount,omitempty"`
		FromUser *string `json:"fromUser,omitempty"`
	}
	var sent []struct {
		Amount *int    `json:"amount,omitempty"`
		ToUser *string `json:"toUser,omitempty"`
	}
	for _, r := range info.CoinHistory.Received {
		amt := r.Amount
		fu := r.FromUser
		received = append(received, struct {
			Amount   *int    `json:"amount,omitempty"`
			FromUser *string `json:"fromUser,omitempty"`
		}{
			Amount:   &amt,
			FromUser: &fu,
		})
	}
	for _, s := range info.CoinHistory.Sent {
		amt := s.Amount
		tu := s.ToUser
		sent = append(sent, struct {
			Amount *int    `json:"amount,omitempty"`
			ToUser *string `json:"toUser,omitempty"`
		}{
			Amount: &amt,
			ToUser: &tu,
		})
	}

	return InfoResponse{
		Coins:     &info.Coins,
		Inventory: &inv,
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
			Received: &received,
			Sent:     &sent,
		},
	}
}

func ptr(s string) *string {
	return &s
}

func errUnauthorized(msg string) error {
	return echo.NewHTTPError(http.StatusUnauthorized, msg)
}
