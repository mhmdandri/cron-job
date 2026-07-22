package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	TargetURL    string
	BearerToken  string
	CronSchedule string // e.g. "15m", "30m", or cron expression "*/15 * * * *"
	HTTPTimeout  int    // in seconds
}

func LoadConfig() *Config {
	// Load .env if available, ignore error if missing
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	targetURL := getEnv("TARGET_URL", "https://shipment.mohaproject.tech/api/cron/monitor")
	bearerToken := getEnv("BEARER_TOKEN", "KMASLDKMNOINFNKmknsnqiwjij19980jn")
	cronSchedule := getEnv("CRON_SCHEDULE", "15m")

	return &Config{
		Port:         port,
		TargetURL:    targetURL,
		BearerToken:  bearerToken,
		CronSchedule: cronSchedule,
		HTTPTimeout:  30,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
