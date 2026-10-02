package router

import (
	"database/sql"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/router/debug"
	"github.com/musiermoore/oksana-vpn-api/internal/router/telegram"
	_telegram "github.com/musiermoore/oksana-vpn-api/internal/telegram"
)

func StartRouter(db *sql.DB) error {
	r := newRouter(db)

	return r.Run(":8080")
}

func newRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

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

	debug.HealthRoutes(r, db)

	// Initialize authentication dependencies.
	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository)

	// Register authentication endpoints.
	auth.RegisterRoutes(r, authService)
	telegram.AuthRoutes(r, _telegram.NewService(authService))

	return r
}
