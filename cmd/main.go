package main

import (
	"avito-shop/internal/api"
	"avito-shop/internal/config"
	"avito-shop/internal/db"
	"avito-shop/internal/logger"
	"avito-shop/internal/middleware"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbConn.Close()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger(log))

	router.POST("/api/auth", api.AuthHandler(dbConn, cfg.JWTSecret, log))

	protected := router.Group("/api")
	protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret, log))
	{
		protected.GET("/info", api.InfoHandler(dbConn, log))
		protected.POST("/sendCoin", api.SendCoinHandler(dbConn, log))
		protected.GET("/buy/:item", api.BuyHandler(dbConn, log))
	}

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	if err := router.Run(port); err != nil {
		log.Fatal("failed to run server", zap.Error(err))
	}
}
