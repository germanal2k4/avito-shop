package main

import (
	"avito-shop/internal/api"
	"avito-shop/internal/config"
	"avito-shop/internal/db"
	"avito-shop/internal/middleware"
	"avito-shop/internal/service"
	"avito-shop/pkg"
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	db.Migrate(dbConn, "migrations")

	zapLogger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(zapLogger)
	logger := pkg.NewZapLogger(zapLogger)

	authDB := db.NewAuthDB(dbConn)
	coinDB := db.NewCoinInventoryDB(dbConn)

	authService := service.NewAuthService(authDB, logger, cfg.JWTSecret)
	shopService := service.NewShopService(coinDB, logger)

	e := echo.New()
	e.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret, zapLogger))

	handlers := &api.Handlers{
		AuthService: authService,
		ShopService: shopService,
		Logger:      logger,
		JWTSecret:   cfg.JWTSecret,
	}

	api.RegisterHandlers(e, handlers)

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	logger.Info("Starting server", zap.String("port", cfg.ServerPort))
	if err := e.Start(port); err != nil {
		logger.Error("Failed to run server", zap.Error(err))
	}
}
