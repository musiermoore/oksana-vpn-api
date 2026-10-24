package telegramdev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const maxMessageBytes = 3900

type Notifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: strings.TrimSpace(botToken),
		chatID:   strings.TrimSpace(chatID),
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (n *Notifier) NotifyException(ctx context.Context, location string, err error, stack string) error {
	if n == nil || n.botToken == "" || n.chatID == "" || err == nil {
		return nil
	}

	message := fmt.Sprintf("API exception\n%s\n%s", location, err)
	if stack != "" {
		message += "\n\n" + stack
	}
	message = truncateUTF8(message, maxMessageBytes)

	body, err := json.Marshal(struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}{ChatID: n.chatID, Text: message})
	if err != nil {
		return fmt.Errorf("encode Telegram exception message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Telegram exception request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		safeErr := strings.ReplaceAll(err.Error(), n.botToken, "[redacted]")
		return fmt.Errorf("send Telegram exception message: %s", safeErr)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16*1024)).Decode(&result); err != nil {
		return fmt.Errorf("decode Telegram exception response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || !result.OK {
		if result.Description == "" {
			result.Description = resp.Status
		}
		return fmt.Errorf("Telegram rejected exception message: %s", result.Description)
	}
	return nil
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}

	value = value[:limit-len("\n...[truncated]")]
	for !utf8.ValidString(value) {
		_, size := utf8.DecodeLastRuneInString(value)
		value = value[:len(value)-size]
	}
	return value + "\n...[truncated]"
}
