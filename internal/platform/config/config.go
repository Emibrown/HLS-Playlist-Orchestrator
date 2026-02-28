package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Load reads the .env file from the current working directory and sets
// environment variables. If .env does not exist, Load returns an error but
// callers can ignore it and use system env or defaults. Pass one or more paths
// to load from specific files (e.g. ".env"); with no paths, ".env" is used.
func Load(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	return godotenv.Load(paths...)
}

// GetEnv returns the value of the environment variable named by key, or fallback
// if the variable is unset or empty.
func GetEnv(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}

// GetEnvInt returns the integer value of the environment variable named by key,
// or fallback if the variable is unset, empty, or not a valid integer.
func GetEnvInt(key string, fallback int) int {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return fallback
}
