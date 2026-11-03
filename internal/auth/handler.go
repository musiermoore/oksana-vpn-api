package auth

import (
	"errors"
	"log"
	"net/http"
	"regexp"
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
	Login    string `json:"login"`
	Telegram string `json:"telegram"`
	Password string `json:"password" binding:"required"`
}

type SetCredentialsRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

var loginPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,64}$`)

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	login := req.Login
	if login == "" {
		login = req.Telegram
	}
	if strings.TrimSpace(login) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, expiresAt, err := h.service.Login(
		c.Request.Context(),
		strings.ToLower(strings.TrimSpace(login)),
		req.Password,
	)

	if errors.Is(err, ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	if err != nil {
		_ = c.Error(err)
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
		"source":       TokenSourceWeb,
	})
}

func (h *Handler) SetCredentials(c *gin.Context) {
	var req SetCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	login := strings.ToLower(strings.TrimSpace(req.Login))
	if !loginPattern.MatchString(login) || len([]byte(req.Password)) < 8 || len([]byte(req.Password)) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "login must be 3-64 letters, digits, dots, underscores or hyphens; password must be 8-72 bytes"})
		return
	}
	value, exists := c.Get(UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := value.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.service.SetCredentials(c.Request.Context(), userID, login, req.Password); err != nil {
		if errors.Is(err, ErrLoginTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "login is already in use"})
			return
		}
		_ = c.Error(err)
		log.Printf("setting credentials failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Logout(c *gin.Context) {
	token := c.GetString("auth_token")

	if err := h.service.Logout(
		c.Request.Context(),
		token,
	); err != nil {
		_ = c.Error(err)
		log.Printf("logout failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
