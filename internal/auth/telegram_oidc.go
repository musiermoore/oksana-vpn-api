package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	telegramOIDCAuthorizeURL = "https://oauth.telegram.org/auth"
	telegramOIDCTokenURL     = "https://oauth.telegram.org/token"
	telegramOIDCJWKSURL      = "https://oauth.telegram.org/.well-known/jwks.json"
	oidcTransactionLifetime  = 10 * time.Minute
	oidcTicketLifetime       = 2 * time.Minute
)

var (
	ErrOIDCDisabled         = errors.New("telegram OIDC is not configured")
	ErrOIDCTransaction      = errors.New("invalid or expired OIDC transaction")
	ErrOIDCIdentity         = errors.New("invalid OIDC identity")
	ErrOIDCIdentityConflict = errors.New("OIDC identity conflict")
	ErrOIDCTicket           = errors.New("invalid or expired login ticket")
	ErrOIDCProvider         = errors.New("Telegram OIDC provider request failed")
)

// TelegramOIDCConfig contains server-only Telegram credentials and fixed URLs.
type TelegramOIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	ReturnURLs   []string
}

func (c TelegramOIDCConfig) valid() bool {
	if strings.TrimSpace(c.ClientID) == "" || strings.TrimSpace(c.ClientSecret) == "" || !isAbsoluteHTTPS(c.RedirectURI) || len(c.ReturnURLs) == 0 {
		return false
	}
	for _, returnURL := range c.ReturnURLs {
		if !isAbsoluteHTTPS(returnURL) {
			return false
		}
	}
	return true
}

func isAbsoluteHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == ""
}

type oidcTransaction struct {
	Nonce     string
	Verifier  string
	ReturnURL string
	ExpiresAt time.Time
}

type telegramOIDCStore interface {
	CreateOIDCTransaction(context.Context, string, oidcTransaction) error
	ConsumeOIDCTransaction(context.Context, string, time.Time) (oidcTransaction, error)
	ResolveOIDCUser(context.Context, string, string, string, string) (User, error)
	CreateOIDCLoginTicket(context.Context, string, int64, time.Time) error
	ConsumeOIDCLoginTicket(context.Context, string, time.Time) (int64, error)
}

type telegramOIDCTokenExchanger interface {
	Exchange(context.Context, string, string) (string, error)
}

type telegramOIDCIDTokenVerifier interface {
	Verify(context.Context, string, string) (telegramOIDCClaims, error)
}

type TelegramOIDCHandler struct {
	flow *telegramOIDCFlow
}

func (h *TelegramOIDCHandler) AllowedOrigins() []string {
	if h.flow == nil {
		return nil
	}
	origins := make([]string, 0, len(h.flow.config.ReturnURLs))
	seen := make(map[string]struct{})
	for _, raw := range h.flow.config.ReturnURLs {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		origin := u.Scheme + "://" + u.Host
		if _, ok := seen[origin]; !ok {
			origins = append(origins, origin)
			seen[origin] = struct{}{}
		}
	}
	return origins
}

func NewTelegramOIDCHandler(service *Service, config TelegramOIDCConfig) *TelegramOIDCHandler {
	h := &TelegramOIDCHandler{}
	if service == nil || !config.valid() {
		return h
	}
	h.flow = newTelegramOIDCFlow(service, config)
	return h
}

func (h *TelegramOIDCHandler) Start(c *gin.Context) {
	if h.flow == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram OIDC unavailable"})
		return
	}
	returnURL, ok := h.flow.allowedReturnURL(c.Query("return_url"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid return URL"})
		return
	}
	location, err := h.flow.start(c.Request.Context(), returnURL)
	if err != nil {
		_ = c.Error(err)
		log.Printf("telegram OIDC start failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to start Telegram login"})
		return
	}
	parsed, _ := url.Parse(location)
	state := parsed.Query().Get("state")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("telegram_oidc_state", state, int(oidcTransactionLifetime.Seconds()), "/api/v1/auth/telegram/oidc/callback", "", true, true)
	c.Redirect(http.StatusFound, location)
}

func (h *TelegramOIDCHandler) Callback(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	if h.flow == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram OIDC unavailable"})
		return
	}
	state, code := c.Query("state"), c.Query("code")
	stateCookie, cookieErr := c.Cookie("telegram_oidc_state")
	if state == "" || cookieErr != nil || subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Telegram login failed"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("telegram_oidc_state", "", -1, "/api/v1/auth/telegram/oidc/callback", "", true, true)
	if c.Query("error") != "" || code == "" {
		_ = h.flow.consumeState(c.Request.Context(), state)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Telegram login failed"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	ticket, returnURL, err := h.flow.complete(c.Request.Context(), state, code)
	if err != nil {
		_ = c.Error(err)
		log.Printf("telegram OIDC callback failed")
		switch {
		case errors.Is(err, ErrOIDCIdentityConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "Telegram identity conflict"})
		case errors.Is(err, ErrOIDCTransaction), errors.Is(err, ErrOIDCIdentity):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Telegram login failed"})
		case errors.Is(err, ErrOIDCProvider):
			c.JSON(http.StatusBadGateway, gin.H{"error": "Telegram login provider unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to complete Telegram login"})
		}
		return
	}
	// Fragment contents are not sent in HTTP requests or Referer headers.
	u, _ := url.Parse(returnURL)
	u.Fragment = "telegram_oidc_ticket=" + url.QueryEscape(ticket)
	c.Redirect(http.StatusSeeOther, u.String())
}

type oidcTicketRequest struct {
	Ticket string `json:"ticket" binding:"required"`
}

func (h *TelegramOIDCHandler) Exchange(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	if h.flow == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram OIDC unavailable"})
		return
	}
	var req oidcTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, expiresAt, err := h.flow.exchangeTicket(c.Request.Context(), req.Ticket)
	if errors.Is(err, ErrOIDCTicket) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired login ticket"})
		return
	}
	if err != nil {
		_ = c.Error(err)
		log.Printf("telegram OIDC ticket exchange failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to establish session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "Bearer", "expires_at": expiresAt, "source": TokenSourceWeb})
}

type telegramOIDCFlow struct {
	issuer interface {
		CreateToken(context.Context, int64, TokenSource) (string, time.Time, error)
	}
	config    TelegramOIDCConfig
	store     telegramOIDCStore
	exchanger telegramOIDCTokenExchanger
	verifier  telegramOIDCIDTokenVerifier
}

func newTelegramOIDCFlow(service *Service, config TelegramOIDCConfig) *telegramOIDCFlow {
	return &telegramOIDCFlow{
		issuer:    service,
		config:    config,
		store:     service.repo,
		exchanger: &telegramTokenExchanger{clientID: config.ClientID, clientSecret: config.ClientSecret, redirectURI: config.RedirectURI},
		verifier:  newTelegramIDTokenVerifier(config.ClientID),
	}
}

func (f *telegramOIDCFlow) allowedReturnURL(candidate string) (string, bool) {
	if candidate == "" && len(f.config.ReturnURLs) == 1 {
		candidate = f.config.ReturnURLs[0]
	}
	for _, allowed := range f.config.ReturnURLs {
		if candidate == allowed && isAbsoluteHTTPS(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func (f *telegramOIDCFlow) start(ctx context.Context, returnURL string) (string, error) {
	state, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	stateHash := sha256Hex(state)
	transaction := oidcTransaction{Nonce: nonce, Verifier: verifier, ReturnURL: returnURL, ExpiresAt: time.Now().UTC().Add(oidcTransactionLifetime)}
	if err := f.store.CreateOIDCTransaction(ctx, stateHash, transaction); err != nil {
		return "", err
	}
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {f.config.ClientID}, "redirect_uri": {f.config.RedirectURI},
		"response_type": {"code"}, "scope": {"openid profile"}, "state": {state},
		"nonce": {nonce}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return telegramOIDCAuthorizeURL + "?" + query.Encode(), nil
}

func (f *telegramOIDCFlow) complete(ctx context.Context, state, code string) (string, string, error) {
	tx, err := f.store.ConsumeOIDCTransaction(ctx, sha256Hex(state), time.Now().UTC())
	if err != nil {
		return "", "", err
	}
	idToken, err := f.exchanger.Exchange(ctx, code, tx.Verifier)
	if err != nil {
		return "", "", fmt.Errorf("%w: token exchange", ErrOIDCProvider)
	}
	claims, err := f.verifier.Verify(ctx, idToken, tx.Nonce)
	if err != nil {
		return "", "", err
	}
	user, err := f.store.ResolveOIDCUser(ctx, claims.Subject, claims.TelegramID, claims.Name, claims.Username)
	if err != nil {
		return "", "", err
	}
	ticket, err := randomURLSafe(32)
	if err != nil {
		return "", "", err
	}
	if err := f.store.CreateOIDCLoginTicket(ctx, sha256Hex(ticket), user.ID, time.Now().UTC().Add(oidcTicketLifetime)); err != nil {
		return "", "", err
	}
	return ticket, tx.ReturnURL, nil
}

func (f *telegramOIDCFlow) consumeState(ctx context.Context, state string) error {
	_, err := f.store.ConsumeOIDCTransaction(ctx, sha256Hex(state), time.Now().UTC())
	return err
}

func (f *telegramOIDCFlow) exchangeTicket(ctx context.Context, ticket string) (string, time.Time, error) {
	if len(ticket) < 40 || len(ticket) > 128 {
		return "", time.Time{}, ErrOIDCTicket
	}
	userID, err := f.store.ConsumeOIDCLoginTicket(ctx, sha256Hex(ticket), time.Now().UTC())
	if err != nil {
		return "", time.Time{}, ErrOIDCTicket
	}
	return f.issuer.CreateToken(ctx, userID, TokenSourceWeb)
}

func randomURLSafe(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
