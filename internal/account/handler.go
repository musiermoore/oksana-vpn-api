package account

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Me(c *gin.Context) {
	value, exists := c.Get(auth.UserIDContextKey)
	userID, ok := value.(int64)
	if !exists || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		log.Printf("load account profile failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
