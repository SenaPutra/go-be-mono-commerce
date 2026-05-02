package config

import "os"

type Config struct {
	AppEnv          string
	HTTPPort        string
	DBDSN           string
	JWTSecret       string
	PaymentProvider string
	CorsAllowOrigin string
	SeedAdminEmail  string
	SeedAdminPass   string
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Load() Config {
	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		DBDSN:           getEnv("DB_DSN", "host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me"),
		PaymentProvider: getEnv("PAYMENT_PROVIDER", "midtrans"),
		CorsAllowOrigin: getEnv("CORS_ALLOW_ORIGIN", "*"),
		SeedAdminEmail:  getEnv("SEED_ADMIN_EMAIL", "admin@example.com"),
		SeedAdminPass:   getEnv("SEED_ADMIN_PASSWORD", "Admin123!"),
	}
}
