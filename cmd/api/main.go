package main

import (
	"log"
	"net/http"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/config"
	"github.com/flexfence/flexfence-backend/internal/db"
	apphttp "github.com/flexfence/flexfence-backend/internal/http"
	"github.com/flexfence/flexfence-backend/internal/jobs"
	"github.com/flexfence/flexfence-backend/internal/mail"
	"github.com/flexfence/flexfence-backend/internal/notify"
	"github.com/flexfence/flexfence-backend/internal/push"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

func main() {
	cfg := config.Load()

	gormDB, err := db.ConnectMySQL(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	if cfg.DBAutoMigrate {
		if err := db.AutoMigrate(gormDB); err != nil {
			log.Fatalf("db migration failed: %v", err)
		}
	}

	tokens, err := auth.NewTokenService(cfg.JWTSecret, cfg.JWTExpiry())
	if err != nil {
		log.Fatalf("jwt init failed: %v", err)
	}

	dataStore := mysqlstore.New(gormDB)
	identityStore := mysqlstore.NewIdentityStore(gormDB)
	mailer := mail.NewFromConfig(mail.SMTPConfig{
		Host:      cfg.SMTPHost,
		Port:      cfg.SMTPPort,
		Username:  cfg.SMTPUser,
		Password:  cfg.SMTPPassword,
		FromEmail: cfg.SMTPFromEmail,
		FromName:  cfg.SMTPFromName,
	})
	if mailer.Enabled() {
		log.Printf("smtp enabled: host=%s port=%d from=%s", cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFromEmail)
	} else {
		log.Printf("smtp not configured; OTP emails will be logged to stdout (set SMTP_HOST and SMTP_FROM_EMAIL)")
	}
	fcmSender := push.NewFCMSender(cfg.FCMProjectID, cfg.FCMServiceAccountFile)
	notifier := notify.NewDispatcher(identityStore, mailer, fcmSender)
	dataStore.SetNotifier(notifier)
	if fcmSender != nil && fcmSender.Enabled() {
		log.Printf("fcm v1 push enabled for project %s", cfg.FCMProjectID)
	} else {
		log.Printf("fcm not configured; set FCM_PROJECT_ID and FCM_SERVICE_ACCOUNT_FILE for mobile push")
	}
	jobs.StartActivityHistoryTrim(dataStore, 24*time.Hour)
	jobs.StartEventGoLiveMonitor(dataStore, time.Minute)

	handler := apphttp.NewRouter(apphttp.RouterDeps{
		DataStore:        dataStore,
		IdentityStore:    identityStore,
		Tokens:           tokens,
		Mailer:           mailer,
		Notifier:         notifier,
		GoogleClient:     cfg.GoogleClientID,
		OTPLength:        cfg.OTPLength,
		OTPExpireMinutes: cfg.OTPExpireMinutes,
		DashboardURL:     cfg.DashboardURL,
		JoinPublicBase:   cfg.JoinPublicBase,
	})
	handler = apphttp.CORSMiddleware(cfg.CORSAllowedOrigins)(handler)

	addr := ":" + cfg.Port
	log.Printf("flexfence-backend starting on %s (%s)", addr, cfg.AppEnv)
	log.Printf("cors allowed origins: %v", cfg.CORSAllowedOrigins)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
