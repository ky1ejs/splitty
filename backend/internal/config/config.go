package config

import "os"

type Config struct {
	Env           string // "development" or "production"
	DatabaseURL   string
	JWTPrivateKey string
}

func Load() Config {
	env := os.Getenv("SPLITTY_ENV")
	if env == "" {
		env = "development"
	}
	return Config{
		Env:           env,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTPrivateKey: os.Getenv("JWT_PRIVATE_KEY"),
	}
}
