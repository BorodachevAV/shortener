package config

import (
	"os"
)

type ShortenerConfig struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DataBaseDNS     string
}

type Config struct {
	Cfg ShortenerConfig
}

func New() *Config {
	return &Config{
		Cfg: ShortenerConfig{
			ServerAddress:   getEnv("SERVER_ADDRESS", ""),
			BaseURL:         getEnv("BASE_URL", ""),
			FileStoragePath: getEnv("FILE_STORAGE_PATH", ""),
			DataBaseDNS:     getEnv("DATABASE_DSN", "")},
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
