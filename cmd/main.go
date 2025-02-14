package main

import (
	"avito-shop/internal/api"
	"avito-shop/internal/config"
	"avito-shop/internal/db"
	"avito-shop/internal/middleware"
	"avito-shop/internal/service"
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func main() {
	// Подгружаем конфиг который будем использовать для подключения с БД
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Подключаемся и пингуем чтобы проверить жива ли она
	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()
	// Катим миграци/
	db.Migrate(dbConn, "migrations")
	// Инициализруем логгер
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("Failed to initialize logger: %v", err)
		}
	}(logger)
	// создаем утилиту
	e := echo.New()
	// прокидываем в утилиту нашу миддлвару
	e.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret, logger))
	// инициализируем апишку
	apiImpl := service.NewEchoAPI(dbConn, cfg.JWTSecret, logger)
	// запускамем хэндлеры
	api.RegisterHandlers(e, apiImpl)

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	logger.Info("Starting server", zap.String("port", port))
	// инициализируем сервис
	if err := e.Start(port); err != nil {
		logger.Fatal("Failed to run server", zap.Error(err))
	}
}
