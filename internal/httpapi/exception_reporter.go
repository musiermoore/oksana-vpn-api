package httpapi

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/telegramdev"
)

func reportExceptions(notifier *telegramdev.Notifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("panic: %v", recovered)
				reportTelegramException(notifier, c, err, string(debug.Stack()))
				c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
			}
		}()

		c.Next()
		for _, requestErr := range c.Errors {
			reportTelegramException(notifier, c, requestErr.Err, "")
		}
		if c.Writer.Status() >= 500 && len(c.Errors) == 0 {
			err := fmt.Errorf("request returned HTTP %d", c.Writer.Status())
			reportTelegramException(notifier, c, err, "")
		}
	}
}

func reportTelegramException(notifier *telegramdev.Notifier, c *gin.Context, err error, stack string) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	location := c.Request.Method + " " + path
	if sendErr := notifier.NotifyException(ctx, location, err, stack); sendErr != nil {
		log.Printf("failed to send Telegram exception notification: %v", sendErr)
	}
}
