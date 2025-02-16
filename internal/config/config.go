package config

import (
	"os"
)

type Config struct {
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	ServerPort       string
	JWTSecret        string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		DatabaseHost:     getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnv("DATABASE_PORT", "5432"),
		DatabaseUser:     getEnv("DATABASE_USER", "postgres"),
		DatabasePassword: getEnv("DATABASE_PASSWORD", "password"),
		DatabaseName:     getEnv("DATABASE_NAME", "shop"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		JWTSecret:        getEnv("JWT_SECRET", "secret"),
	}
	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
