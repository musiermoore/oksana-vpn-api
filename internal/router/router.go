package router

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/router/debug"
)

func StartRouter(db *sql.DB) error {
	r := newRouter(db)

	return r.Run(":8080")
}

func newRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	debug.HealthRoutes(r, db)

	// Initialize authentication dependencies.
	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository)

	// Register authentication endpoints.
	auth.RegisterRoutes(r, authService)

	return r
}
