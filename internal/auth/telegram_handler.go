package auth

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/telegram"
)

type telegramLoginRequest struct {
	InitData string `json:"init_data" binding:"required"`
}

func (h *Handler) Telegram(c *gin.Context) {
	var req telegramLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	data, err := telegram.ParseInitData(req.InitData)
	if err != nil {
		log.Printf("telegram auth parse error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid data"})
		return
	}
	isValid, err := telegram.ValidateInitData(data, h.telegramBotToken)
	if !isValid {
		log.Printf("telegram auth validation error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}
	user, err := h.service.GetOrCreateUserByTelegramID(
		c.Request.Context(),
		strconv.FormatInt(data.User.ID, 10),
		data.User.FirstName,
		data.User.LastName,
		data.User.Username,
	)
	if err != nil {
		_ = c.Error(err)
		log.Printf("telegram auth user create or lookup error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load user"})
		return
	}
	if err := h.service.AttachReferral(c.Request.Context(), user, &data.StartParam); err != nil {
		_ = c.Error(err)
		log.Printf("telegram referral error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to process referral"})
		return
	}
	token, expiresAt, err := h.service.CreateToken(c.Request.Context(), user.ID, TokenSourceTelegram)
	if err != nil {
		_ = c.Error(err)
		log.Printf("telegram token creation error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalid"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"source":      TokenSourceTelegram,
		"expires_at":  expiresAt,
		"id":          user.ID,
		"name":        user.Name,
		"telegram":    user.Telegram,
		"telegram_id": user.TelegramID,
	})
}
