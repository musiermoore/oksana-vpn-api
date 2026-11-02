package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type telegramTokenExchanger struct {
	clientID     string
	clientSecret string
	redirectURI  string
	client       *http.Client
}

func (e *telegramTokenExchanger) Exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {e.redirectURI}, "client_id": {e.clientID}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramOIDCTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(e.clientID, e.clientSecret)
	client := e.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Telegram token request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("Telegram rejected authorization code")
	}
	var body struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil || body.IDToken == "" {
		return "", errors.New("Telegram token response did not include an ID token")
	}
	return body.IDToken, nil
}

type telegramOIDCClaims struct {
	Issuer          string          `json:"iss"`
	Audience        audienceClaim   `json:"aud"`
	Subject         string          `json:"sub"`
	AuthorizedParty string          `json:"azp,omitempty"`
	ExpiresAt       int64           `json:"exp"`
	IssuedAt        int64           `json:"iat"`
	NotBefore       int64           `json:"nbf,omitempty"`
	Nonce           string          `json:"nonce"`
	Name            string          `json:"name"`
	Username        string          `json:"preferred_username"`
	ProfileID       json.RawMessage `json:"id,omitempty"`
	TelegramID      string          `json:"-"`
}

type audienceClaim []string

var errUnknownTelegramSigningKey = errors.New("unknown Telegram signing key")

func oidcIdentityError(reason string) error {
	return fmt.Errorf("%w: %s", ErrOIDCIdentity, reason)
}

func (a *audienceClaim) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*a = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

type telegramIDTokenVerifier struct {
	clientID string
	client   *http.Client
	mu       sync.Mutex
	keys     map[string]*rsa.PublicKey
	loadedAt time.Time
}

const oidcClockSkew = 60 * time.Second

func newTelegramIDTokenVerifier(clientID string) *telegramIDTokenVerifier {
	return &telegramIDTokenVerifier{clientID: clientID, client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (v *telegramIDTokenVerifier) Verify(ctx context.Context, compact, expectedNonce string) (telegramOIDCClaims, error) {
	var claims telegramOIDCClaims
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return claims, oidcIdentityError("malformed ID token")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, oidcIdentityError("malformed ID token header")
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(headerBytes, &header) != nil {
		return claims, oidcIdentityError("malformed ID token header")
	}
	if header.Algorithm != "RS256" {
		return claims, oidcIdentityError("unsupported signing algorithm")
	}
	if header.KeyID == "" {
		return claims, oidcIdentityError("missing signing key identifier")
	}
	key, err := v.key(ctx, header.KeyID, false)
	if err != nil {
		key, err = v.key(ctx, header.KeyID, true)
	}
	if err != nil || key == nil {
		if errors.Is(err, errUnknownTelegramSigningKey) {
			return claims, oidcIdentityError("signing key not found in Telegram JWKS")
		}
		return claims, fmt.Errorf("%w: unable to load Telegram signing keys", ErrOIDCProvider)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, oidcIdentityError("malformed ID token signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
		return claims, oidcIdentityError("signature verification failed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil {
		return claims, oidcIdentityError("malformed ID token claims")
	}
	telegramID, err := parseTelegramOIDCProfileID(claims.ProfileID)
	if err != nil {
		return telegramOIDCClaims{}, oidcIdentityError("invalid Telegram profile ID")
	}
	claims.TelegramID = telegramID
	now := time.Now().Unix()
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == v.clientID {
			validAudience = true
		}
	}
	if claims.Issuer != "https://oauth.telegram.org" {
		return telegramOIDCClaims{}, oidcIdentityError("issuer mismatch")
	}
	if !validAudience || len(claims.Audience) == 0 {
		return telegramOIDCClaims{}, oidcIdentityError("audience mismatch")
	}
	if claims.Subject == "" {
		return telegramOIDCClaims{}, oidcIdentityError("subject is missing")
	}
	if claims.ExpiresAt <= now-int64(oidcClockSkew.Seconds()) {
		return telegramOIDCClaims{}, oidcIdentityError("ID token has expired")
	}
	if claims.IssuedAt <= 0 || claims.IssuedAt > now+60 || claims.ExpiresAt <= claims.IssuedAt {
		return telegramOIDCClaims{}, oidcIdentityError("issued-at claim is invalid")
	}
	if claims.NotBefore != 0 && claims.NotBefore > now+60 {
		return telegramOIDCClaims{}, oidcIdentityError("ID token is not yet valid")
	}
	if claims.Nonce == "" || claims.Nonce != expectedNonce {
		return telegramOIDCClaims{}, oidcIdentityError("nonce mismatch")
	}
	if (claims.AuthorizedParty != "" && claims.AuthorizedParty != v.clientID) ||
		(len(claims.Audience) > 1 && claims.AuthorizedParty != v.clientID) {
		return telegramOIDCClaims{}, oidcIdentityError("authorized-party mismatch")
	}
	return claims, nil
}

func parseTelegramOIDCProfileID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var id json.Number
	if err := json.Unmarshal(raw, &id); err == nil {
		value, err := strconv.ParseUint(id.String(), 10, 64)
		if err != nil || value == 0 {
			return "", errors.New("invalid Telegram profile ID")
		}
		return strconv.FormatUint(value, 10), nil
	}
	var stringID string
	if err := json.Unmarshal(raw, &stringID); err != nil {
		return "", errors.New("invalid Telegram profile ID")
	}
	value, err := strconv.ParseUint(stringID, 10, 64)
	if err != nil || value == 0 {
		return "", errors.New("invalid Telegram profile ID")
	}
	return strconv.FormatUint(value, 10), nil
}

func (v *telegramIDTokenVerifier) key(ctx context.Context, kid string, forceRefresh bool) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !forceRefresh && time.Since(v.loadedAt) < 5*time.Minute {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
		return nil, errUnknownTelegramSigningKey
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, telegramOIDCJWKSURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Telegram JWKS request failed")
	}
	var document struct {
		Keys []struct {
			KeyType   string `json:"kty"`
			Use       string `json:"use"`
			Algorithm string `json:"alg"`
			KeyID     string `json:"kid"`
			Modulus   string `json:"n"`
			Exponent  string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&document); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range document.Keys {
		if jwk.KeyType != "RSA" || (jwk.Use != "" && jwk.Use != "sig") || (jwk.Algorithm != "" && jwk.Algorithm != "RS256") || jwk.KeyID == "" {
			continue
		}
		n, nErr := base64.RawURLEncoding.DecodeString(jwk.Modulus)
		e, eErr := base64.RawURLEncoding.DecodeString(jwk.Exponent)
		if nErr != nil || eErr != nil || len(e) == 0 || len(e) > 4 || new(big.Int).SetBytes(n).BitLen() < 2048 {
			continue
		}
		exponent := 0
		for _, b := range e {
			exponent = exponent<<8 | int(b)
		}
		if exponent < 3 || exponent%2 == 0 {
			continue
		}
		keys[jwk.KeyID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
	}
	if len(keys) == 0 {
		return nil, errors.New("Telegram JWKS contained no supported signing keys")
	}
	v.keys, v.loadedAt = keys, time.Now()
	if key := v.keys[kid]; key != nil {
		return key, nil
	}
	return nil, errUnknownTelegramSigningKey
}
