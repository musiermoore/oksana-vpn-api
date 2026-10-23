package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const initDataMaxAge = 24 * time.Hour
const initDataFutureSkew = 2 * time.Minute

type User struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm"`
}

type InitData struct {
	QueryID    string
	User       User
	AuthDate   int64
	StartParam string
	Signature  string
	Hash       string
	Raw        url.Values
}

func ParseInitData(data string) (InitData, error) {
	values, err := url.ParseQuery(data)
	if err != nil {
		return InitData{}, err
	}
	var user User
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil {
		return InitData{}, err
	}
	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return InitData{}, err
	}
	return InitData{
		QueryID:    values.Get("query_id"),
		User:       user,
		AuthDate:   authDate,
		StartParam: values.Get("start_param"),
		Signature:  values.Get("signature"),
		Hash:       values.Get("hash"),
		Raw:        values,
	}, nil
}

func ValidateInitData(data InitData, botToken string) (bool, error) {
	authTime := time.Unix(data.AuthDate, 0).UTC()
	now := time.Now().UTC()
	if authTime.After(now.Add(initDataFutureSkew)) || now.Sub(authTime) > initDataMaxAge {
		return false, fmt.Errorf("telegram init data has an invalid date")
	}

	values := data.Raw
	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return false, fmt.Errorf("telegram init data has no hash")
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
	secretKey := hmacHash([]byte("WebAppData"), []byte(botToken))
	calculatedHash := hmacHash(secretKey, []byte(dataCheckString))
	received, err := hex.DecodeString(receivedHash)
	if err != nil {
		return false, fmt.Errorf("decode telegram init data hash: %w", err)
	}
	return hmac.Equal(calculatedHash, received), nil
}

func hmacHash(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
