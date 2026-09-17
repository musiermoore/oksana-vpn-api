package debug

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
}
