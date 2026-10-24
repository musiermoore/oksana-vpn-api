package httpapi

import (
	"database/sql"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/account"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/telegramdev"
)

func NewRouter(
	db *sql.DB,
	authHandler *auth.Handler,
	accountHandler *account.Handler,
	authService *auth.Service,
	notifiers ...*telegramdev.Notifier,
) *gin.Engine {
	var notifier *telegramdev.Notifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}
	r := gin.New()
	r.Use(gin.Logger(), reportExceptions(notifier))
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://panel.oksana1984.ru",
			"https://public.oksana1984.ru",
			"https://oksana1984.ru",
		},
		AllowMethods: []string{
			"GET",
			"POST",
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

	protected := authRoutes.Group("")
	protected.Use(auth.RequireAuth(authService))
	protected.GET("/me", accountHandler.Me)
	protected.POST("/logout", authHandler.Logout)
	return r
}
