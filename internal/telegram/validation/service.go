package validation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/musiermoore/oksana-vpn-api/internal/telegram/types"
)

const telegramInitDataMaxAge = 24 * time.Hour
const telegramInitDataFutureSkew = 2 * time.Minute

type TelegramService struct {
	InitData types.TelegramInitData
}

func Service(data types.TelegramInitData) TelegramService {
	return TelegramService{
		InitData: data,
	}
}

type ValidationInitData struct {
	QueryId  string             `json:"query_id"`
	User     types.TelegramUser `json:"user"`
	AuthDate int64              `json:"auth_date"`
}

func (service *TelegramService) IsValid() (bool, error) {
	isDateValid := service.isDateValid()

	if !isDateValid {
		return false, fmt.Errorf("Telegram Validation Init Data Error: Invalid date")
	}

	values := service.InitData.Raw

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return false, fmt.Errorf("Telegram Validation Init Data Error: Invalid Received Hash")
	}

	values.Del("hash")

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}

	dataCheckString := strings.Join(parts, "\n")

	secretKey := service.getHash(
		[]byte("WebAppData"),
		[]byte(service.GetBotToken()),
	)

	calculatedHash := service.getHash(
		secretKey,
		[]byte(dataCheckString),
	)

	received, err := hex.DecodeString(receivedHash)
	if err != nil {
		return false, fmt.Errorf("Telegram Validation Init Data Error: Decode received")
	}

	return hmac.Equal(
		calculatedHash,
		received,
	), nil
}

func (service *TelegramService) GetBotToken() string {
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

func (service *TelegramService) getHash(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)

	return mac.Sum(nil)
}

func (service *TelegramService) isDateValid() bool {
	authTime := time.Unix(service.InitData.AuthDate, 0).UTC()
	now := time.Now().UTC()

	if authTime.After(now.Add(telegramInitDataFutureSkew)) {
		return false
	}

	return now.Sub(authTime) <= telegramInitDataMaxAge
}
