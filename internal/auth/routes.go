package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	router *gin.Engine,
	service *Service,
) {
	handler := NewHandler(service)

	auth := router.Group("/api/v1/auth")

	// Public
	auth.POST("/login", handler.Login)

	// Protected
	protected := auth.Group("")
	protected.Use(RequireAuth(service))

	protected.GET("/me", handler.Me)
	protected.POST("/logout", handler.Logout)
}
