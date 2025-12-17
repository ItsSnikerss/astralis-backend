package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DBUser          string
	DBPass          string
	DBHost          string
	DBName          string
	JWTSecret       []byte
	Port            string
	TurnstileSecret string
	ResendApiKey    string
	EmailSender     string
}

var Cfg *AppConfig

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	Cfg = &AppConfig{
		DBUser:          os.Getenv("DB_USER"),
		DBPass:          os.Getenv("DB_PASSWORD"),
		DBHost:          os.Getenv("DB_HOST"),
		DBName:          os.Getenv("DB_NAME"),
		JWTSecret:       []byte(os.Getenv("JWT_SECRET_KEY")),
		Port:            os.Getenv("SERVER_PORT"),
		TurnstileSecret: os.Getenv("TURNSTILE_SECRET_KEY"),
		ResendApiKey:    os.Getenv("RESEND_API_KEY"),
		EmailSender:     os.Getenv("EMAIL_SENDER"),
	}

	if Cfg.Port == "" {
		Cfg.Port = ":8080"
	}
}