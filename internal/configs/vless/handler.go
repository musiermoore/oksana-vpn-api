package vless

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
)

const (
	defaultPublicDomain  = "https://connect.oksana1984.ru"
	accessPaymentMessage = "Нет активной подписки. Откройте раздел оплаты и активируйте доступ."
	internalErrorMessage = "Что-то пошло не так 🤯\n\nПопробуй ещё раз позже или обратись в поддержку."
)

type Handler struct {
	service *Service
}

func NewHandler(db *sql.DB) *Handler {
	publicDomain := strings.TrimRight(strings.TrimSpace(os.Getenv("VLESS_PUBLIC_DOMAIN")), "/")
	if publicDomain == "" {
		publicDomain = defaultPublicDomain
	}
	return &Handler{service: NewService(NewRepository(db), publicDomain)}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, requireAuth gin.HandlerFunc) {
	api := router.Group("/api/v1")
	if requireAuth != nil {
		api.Use(requireAuth)
	}
	api.GET("/vless/link", h.Get)
}

func (h *Handler) Get(c *gin.Context) {
	value, exists := c.Get(auth.UserIDContextKey)
	userID, ok := value.(int64)
	if !exists || !ok || userID < 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	links, err := h.service.Links(c.Request.Context(), userID)

	if errors.Is(err, ErrNoActiveAccess) {
		c.JSON(http.StatusForbidden, gin.H{"type": "debt", "message": accessPaymentMessage})
		return
	}

	if err != nil {
		log.Printf("build VLESS links for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": internalErrorMessage})
		return
	}

	c.JSON(http.StatusOK, links)
}
