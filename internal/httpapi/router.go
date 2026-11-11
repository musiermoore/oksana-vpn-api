package httpapi

import (
	"database/sql"
	"slices"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/account"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/configs/vless"
	"github.com/musiermoore/oksana-vpn-api/internal/telegramdev"
)

func NewRouter(
	db *sql.DB,
	authHandler *auth.Handler,
	accountHandler *account.Handler,
	authService *auth.Service,
	notifiers ...*telegramdev.Notifier,
) *gin.Engine {
	return NewRouterWithOIDC(db, authHandler, accountHandler, authService, nil, notifiers...)
}

func NewRouterWithOIDC(
	db *sql.DB,
	authHandler *auth.Handler,
	accountHandler *account.Handler,
	authService *auth.Service,
	oidcHandler *auth.TelegramOIDCHandler,
	notifiers ...*telegramdev.Notifier,
) *gin.Engine {
	var notifier *telegramdev.Notifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}
	r := gin.New()
	r.Use(gin.Logger(), reportExceptions(notifier))
	allowOrigins := []string{
		"https://panel.oksana1984.ru",
		"https://public.oksana1984.ru",
		"https://oksana1984.ru",
	}
	if oidcHandler != nil {
		allowOrigins = append(allowOrigins, oidcHandler.AllowedOrigins()...)
	}
	slices.Sort(allowOrigins)
	allowOrigins = slices.Compact(allowOrigins)
	r.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Client-Timezone",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	registerHealthRoutes(r, db)

	authRoutes := r.Group("/api/v1/auth")
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/telegram", authHandler.Telegram)
	if oidcHandler == nil {
		oidcHandler = auth.NewTelegramOIDCHandler(nil, auth.TelegramOIDCConfig{})
	}
	authRoutes.GET("/telegram/oidc/start", oidcHandler.Start)
	authRoutes.GET("/telegram/oidc/callback", oidcHandler.Callback)
	authRoutes.POST("/telegram/oidc/exchange", oidcHandler.Exchange)

	protected := authRoutes.Group("")
	protected.Use(auth.RequireAuth(authService))
	protected.GET("/me", accountHandler.Me)
	protected.POST("/logout", authHandler.Logout)
	protected.PUT("/credentials", authHandler.SetCredentials)

	vless.NewHandler(db).RegisterRoutes(r, auth.RequireAuth(authService))
	return r
}
