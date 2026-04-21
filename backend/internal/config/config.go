package config

import "os"

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	Env           string // EnvDevelopment or EnvProduction
	DatabaseURL   string
	JWTPrivateKey string
}

func Load() Config {
	env := os.Getenv("SPLITTY_ENV")
	if env == "" {
		env = EnvDevelopment
	}
	return Config{
		Env:           env,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTPrivateKey: os.Getenv("JWT_PRIVATE_KEY"),
	}
}
