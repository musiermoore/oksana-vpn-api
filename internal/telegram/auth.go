package telegram

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
	"github.com/musiermoore/oksana-vpn-api/internal/telegram/types"
	"github.com/musiermoore/oksana-vpn-api/internal/telegram/validation"
)

type TelegramService struct {
	AuthService *auth.Service
}

func NewService(authService *auth.Service) TelegramService {
	return TelegramService{AuthService: authService}
}

func (service *TelegramService) AuthorizeByInitData(c *gin.Context) {
	var req types.TelegramAuthRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	data, err := parseInitData(req.InitData)

	if err != nil {
		fmt.Println("Telegarm Auth Error: ", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid data",
		})

		return
	}

	validationService := validation.Service(data)
	isValid, validErr := validationService.IsValid()

	if !isValid {
		fmt.Println("Telegarm Auth Error: ", validErr)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user",
		})

		return
	}

	user, err := service.AuthService.GetUserByTelegramId(c, strconv.FormatInt(data.User.ID, 10))

	if err != nil {
		fmt.Println("Telegarm Auth Error: ", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not found",
		})

		return
	}

	token, expiresAt, err := service.AuthService.CreateToken(c, user)

	if err != nil {
		fmt.Println("Telegarm Auth Error: ", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token invalid",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       true,
		"access_token": token,
		"expires_at":   expiresAt,
	})
}
