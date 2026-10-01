package auth

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

		user, err := service.Authenticate(
			c.Request.Context(),
			token,
		)

		fmt.Println(user, err)

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

		c.Set("auth_user", user)
		c.Set("auth_token", token)

		c.Next()
	}
}
