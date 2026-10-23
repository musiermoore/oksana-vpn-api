package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const UserIDContextKey = "auth_user_id"

func RequireAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(
			c.GetHeader("Authorization"),
		)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "unauthorized"},
			)
			return
		}

		token := parts[1]

		userID, err := service.Authenticate(
			c.Request.Context(),
			token,
		)

		if errors.Is(err, ErrInvalidToken) {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "unauthorized"},
			)
			return
		}

		if err != nil {
			log.Printf("authentication database error: %v", err)

			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				gin.H{"error": "service unavailable"},
			)
			return
		}

		c.Set(UserIDContextKey, userID)
		c.Set("auth_token", token)

		c.Next()
	}
}
