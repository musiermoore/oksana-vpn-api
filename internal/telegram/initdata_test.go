package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestParseAndValidateInitData(t *testing.T) {
	const botToken = "123456:example"
	authDate := strconv.FormatInt(time.Now().Unix(), 10)
	user := `{"id":42,"first_name":"Oksana"}`
	checkString := "auth_date=" + authDate + "\nuser=" + user
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	signature := hmac.New(sha256.New, secret.Sum(nil))
	signature.Write([]byte(checkString))

	values := url.Values{
		"auth_date": {authDate},
		"user":      {user},
		"hash":      {hex.EncodeToString(signature.Sum(nil))},
	}
	data, err := ParseInitData(values.Encode())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := ValidateInitData(data, botToken)
	if err != nil || !valid {
		t.Fatalf("valid Telegram init data rejected: valid=%v, err=%v", valid, err)
	}
	if data.User.ID != 42 {
		t.Fatalf("parsed user ID = %d; want 42", data.User.ID)
	}

	values.Set("user", `{"id":43,"first_name":"Oksana"}`)
	tampered, err := ParseInitData(values.Encode())
	if err != nil {
		t.Fatal(err)
	}
	valid, err = ValidateInitData(tampered, botToken)
	if err != nil || valid {
		t.Fatalf("tampered Telegram init data accepted: valid=%v, err=%v", valid, err)
	}
}
