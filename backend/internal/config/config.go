package config

import (
	"os"
	"strings"
)

// Config holds process-level settings read from the environment.
// Settings that users change at runtime (refresh interval, retention)
// live in the database instead.
type Config struct {
	Addr     string
	DataDir  string
	Password string
}

func Load() Config {
	return Config{
		Addr:     env("PIKAPU_ADDR", ":7660"),
		DataDir:  env("PIKAPU_DATA_DIR", "./data"),
		Password: os.Getenv("PIKAPU_PASSWORD"),
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
