package telegram

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/musiermoore/oksana-vpn-api/internal/telegram/types"
)

func parseInitData(data string) (types.TelegramInitData, error) {
	values, err := url.ParseQuery(data)
	if err != nil {
		return types.TelegramInitData{}, err
	}

	var user types.TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil {
		return types.TelegramInitData{}, err
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return types.TelegramInitData{}, err
	}

	return types.TelegramInitData{
		QueryId:   values.Get("query_id"),
		User:      user,
		AuthDate:  authDate,
		Signature: values.Get("signature"),
		Hash:      values.Get("hash"),
		Raw:       values,
	}, nil
}
