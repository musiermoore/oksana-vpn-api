package telegram

import (
	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/telegram"
)

func AuthRoutes(router *gin.Engine, service telegram.TelegramService) {
	router.POST("/api/v1/auth/telegram", service.AuthorizeByInitData)
}
