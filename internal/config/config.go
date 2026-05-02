package config

import "os"

type Config struct {
	AppEnv               string
	HTTPPort             string
	DBDSN                string
	JWTSecret            string
	JWTTTLHours          string
	PaymentProvider      string
	PaymentMockMode      bool
	MidtransServerKey    string
	MidtransClientKey    string
	MidtransIsProduction bool
	XenditSecretKey      string
	XenditCallbackToken  string
	CorsAllowOrigin      string
	SeedAdminEmail       string
	SeedAdminPass        string
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Load() Config {
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("HTTP_PORT", "8080"),
		DBDSN:                getEnv("DB_DSN", "host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable"),
		JWTSecret:            getEnv("JWT_SECRET", "change-me"),
		JWTTTLHours:          getEnv("JWT_TTL_HOURS", "24"),
		PaymentProvider:      getEnv("PAYMENT_PROVIDER", "midtrans"),
		PaymentMockMode:      getEnv("PAYMENT_MOCK_MODE", "true") == "true",
		MidtransServerKey:    getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey:    getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransIsProduction: getEnv("MIDTRANS_IS_PRODUCTION", "false") == "true",
		XenditSecretKey:      getEnv("XENDIT_SECRET_KEY", ""),
		XenditCallbackToken:  getEnv("XENDIT_CALLBACK_TOKEN", ""),
		CorsAllowOrigin:      getEnv("CORS_ALLOW_ORIGIN", "*"),
		SeedAdminEmail:       getEnv("SEED_ADMIN_EMAIL", "admin@example.com"),
		SeedAdminPass:        getEnv("SEED_ADMIN_PASSWORD", "Admin123!"),
	}
}
