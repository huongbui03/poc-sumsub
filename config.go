package main

import (
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"log"
	"os"
)

type SumsubConfig struct {
	AppToken        string
	SecretKey       string
	LevelName       string
	WebhookEndpoint string // Your server's webhook endpoint
	WebhookSecret   string // Optional secret to verify webhook signature (or payload digest)
}

func loadConfig() (*SumsubConfig, error) {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	appToken := os.Getenv("SUMSUB_APP_TOKEN")
	secretKey := os.Getenv("SUMSUB_SECRET_KEY")
	levelName := os.Getenv("SUMSUB_LEVEL_NAME")
	webhookEndpoint := os.Getenv("SUMSUB_WEBHOOK_ENDPOINT")
	webhookSecret := os.Getenv("SUMSUB_WEBHOOK_SECRET")
	// Document path is no longer needed in the config as the server will receive documents

	if appToken == "" || secretKey == "" || webhookEndpoint == "" {
		return nil, errors.New("SUMSUB_APP_TOKEN, SUMSUB_SECRET_KEY, and SUMSUB_WEBHOOK_ENDPOINT must be set in environment variables")
	}

	if levelName == "" {
		levelName = "basic-kyc-level" // Default level if not set
	}

	return &SumsubConfig{
		AppToken:        appToken,
		SecretKey:       secretKey,
		LevelName:       levelName,
		WebhookEndpoint: webhookEndpoint,
		WebhookSecret:   webhookSecret,
	}, nil
}
