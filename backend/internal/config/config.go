package config

import "os"

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	Env               string // EnvDevelopment or EnvProduction
	DatabaseURL       string
	JWTPrivateKey     string
	CORSAllowedOrigin string
	MailgunAPIKey     string
	MailgunDomain     string
	MailgunFrom       string
}

func Load() Config {
	env := os.Getenv("SPLITTY_ENV")
	if env == "" {
		env = EnvDevelopment
	}
	corsOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if corsOrigin == "" && env == EnvDevelopment {
		corsOrigin = "http://localhost:5173"
	}

	return Config{
		Env:               env,
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTPrivateKey:     os.Getenv("JWT_PRIVATE_KEY"),
		CORSAllowedOrigin: corsOrigin,
		MailgunAPIKey:     os.Getenv("MAILGUN_API_KEY"),
		MailgunDomain:     os.Getenv("MAILGUN_DOMAIN"),
		MailgunFrom:       os.Getenv("MAILGUN_FROM"),
	}
}
