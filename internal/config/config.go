package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	AppEnv        string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBAutoMigrate  bool
	JWTSecret      string
	JWTExpireHours int
	GoogleClientID     string
	CORSAllowedOrigins []string
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPassword       string
	SMTPFromEmail      string
	SMTPFromName       string
	OTPLength          int
	OTPExpireMinutes   int
	DashboardURL       string
	JoinPublicBase          string
	FCMProjectID            string
	FCMServiceAccountFile   string
}

func Load() Config {
	// Load .env for local development; existing environment variables remain authoritative.
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "flexfence"
	}
	dbAutoMigrate := parseBoolEnv("DB_AUTO_MIGRATE")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-only-change-me"
	}
	jwtExpireHours := 72
	if raw := os.Getenv("JWT_EXPIRE_HOURS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			jwtExpireHours = parsed
		}
	}

	corsOrigins := parseCSVEnv("CORS_ALLOWED_ORIGINS")
	if len(corsOrigins) == 0 {
		corsOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	smtpPort := 587
	if raw := os.Getenv("SMTP_PORT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			smtpPort = parsed
		}
	}

	otpLength := 4
	if raw := os.Getenv("OTP_LENGTH"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 4 && parsed <= 8 {
			otpLength = parsed
		}
	}
	otpExpireMinutes := 10
	if raw := os.Getenv("OTP_EXPIRE_MINUTES"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			otpExpireMinutes = parsed
		}
	}

	dashboardURL := strings.TrimSpace(os.Getenv("DASHBOARD_URL"))
	if dashboardURL == "" {
		dashboardURL = "http://localhost:5173"
	}

	return Config{
		Port:               port,
		AppEnv:             appEnv,
		DBHost:             dbHost,
		DBPort:             dbPort,
		DBUser:             dbUser,
		DBPassword:         dbPassword,
		DBName:             dbName,
		DBAutoMigrate:      dbAutoMigrate,
		JWTSecret:          jwtSecret,
		JWTExpireHours:     jwtExpireHours,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		CORSAllowedOrigins: corsOrigins,
		SMTPHost:           strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:           smtpPort,
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPFromEmail:      strings.TrimSpace(os.Getenv("SMTP_FROM_EMAIL")),
		SMTPFromName:       strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
		OTPLength:          otpLength,
		OTPExpireMinutes:   otpExpireMinutes,
		DashboardURL:       dashboardURL,
		JoinPublicBase:        strings.TrimSpace(os.Getenv("JOIN_PUBLIC_BASE_URL")),
		FCMProjectID:          strings.TrimSpace(os.Getenv("FCM_PROJECT_ID")),
		FCMServiceAccountFile: strings.TrimSpace(os.Getenv("FCM_SERVICE_ACCOUNT_FILE")),
	}
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (c Config) JWTExpiry() time.Duration {
	return time.Duration(c.JWTExpireHours) * time.Hour
}

func parseBoolEnv(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return value == "true" || value == "1" || value == "yes"
}
