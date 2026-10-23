package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service          *Service
	telegramBotToken string
}

func NewHandler(service *Service, telegramBotToken string) *Handler {
	return &Handler{service: service, telegramBotToken: telegramBotToken}
}

type LoginRequest struct {
	Telegram string `json:"telegram" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	token, expiresAt, err := h.service.Login(
		c.Request.Context(),
		strings.ToLower(strings.TrimSpace(req.Telegram)),
		req.Password,
	)

	if errors.Is(err, ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	if err != nil {
		log.Printf("login failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expiresAt,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	token := c.GetString("auth_token")

	if err := h.service.Logout(
		c.Request.Context(),
		token,
	); err != nil {
		log.Printf("logout failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
