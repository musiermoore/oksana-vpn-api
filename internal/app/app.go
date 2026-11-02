package app

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/musiermoore/oksana-vpn-api/internal/account"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/database"
	"github.com/musiermoore/oksana-vpn-api/internal/httpapi"
	"github.com/musiermoore/oksana-vpn-api/internal/referral"
	"github.com/musiermoore/oksana-vpn-api/internal/telegramdev"
)

func Run() error {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system environment variables")
	}

	dsn, err := database.BuildDSN(
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := database.NewMySQL(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Println("MySQL connected successfully")

	// sql.DB is one shared connection pool; each repository keeps a reference to it.
	referralRepository := referral.NewRepository(db)
	authRepository := auth.NewRepository(db)
	accountRepository := account.NewRepository(db)

	referrals := referral.NewService(referralRepository, os.Getenv("TELEGRAM_BOT_USERNAME"))
	authService := auth.NewService(authRepository, referrals)
	accountService := account.NewService(accountRepository, referrals)
	authHandler := auth.NewHandler(authService, os.Getenv("TELEGRAM_BOT_TOKEN"))
	returnURLs := splitConfigURLs(os.Getenv("TELEGRAM_OIDC_FRONTEND_RETURN_URLS"))
	oidcHandler := auth.NewTelegramOIDCHandler(authService, auth.TelegramOIDCConfig{
		ClientID:     os.Getenv("TELEGRAM_OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("TELEGRAM_OIDC_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("TELEGRAM_OIDC_REDIRECT_URI"),
		ReturnURLs:   returnURLs,
	})
	accountHandler := account.NewHandler(accountService)
	exceptionNotifier := telegramdev.NewNotifier(
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_DEV_CHAT_ID"),
	)
	router := httpapi.NewRouterWithOIDC(
		db,
		authHandler,
		accountHandler,
		authService,
		oidcHandler,
		exceptionNotifier,
	)
	return router.Run(":8080")
}

func splitConfigURLs(value string) []string {
	var urls []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			urls = append(urls, item)
		}
	}
	return urls
}
