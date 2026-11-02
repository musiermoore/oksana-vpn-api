package auth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeOIDCStore struct {
	txns         map[string]oidcTransaction
	users        map[string]User
	tickets      map[string]int64
	provisions   int
	provisionErr error
}

func newFakeOIDCStore() *fakeOIDCStore {
	return &fakeOIDCStore{txns: map[string]oidcTransaction{}, users: map[string]User{}, tickets: map[string]int64{}}
}
func (s *fakeOIDCStore) CreateOIDCTransaction(_ context.Context, hash string, tx oidcTransaction) error {
	s.txns[hash] = tx
	return nil
}
func (s *fakeOIDCStore) ConsumeOIDCTransaction(_ context.Context, hash string, now time.Time) (oidcTransaction, error) {
	tx, ok := s.txns[hash]
	if !ok || !tx.ExpiresAt.After(now) {
		delete(s.txns, hash)
		return oidcTransaction{}, ErrOIDCTransaction
	}
	delete(s.txns, hash)
	return tx, nil
}
func (s *fakeOIDCStore) FindUserByOIDCSubject(_ context.Context, subject string) (User, error) {
	user, ok := s.users[subject]
	if !ok {
		return User{}, ErrOIDCIdentity
	}
	return user, nil
}
func (s *fakeOIDCStore) ProvisionOIDCUser(_ context.Context, subject, name, _ string) (User, error) {
	s.provisions++
	if s.provisionErr != nil {
		return User{}, s.provisionErr
	}
	if user, ok := s.users[subject]; ok {
		return user, nil
	}
	user := User{ID: int64(len(s.users) + 1), Name: name}
	s.users[subject] = user
	return user, nil
}
func (s *fakeOIDCStore) CreateOIDCLoginTicket(_ context.Context, hash string, userID int64, _ time.Time) error {
	s.tickets[hash] = userID
	return nil
}
func (s *fakeOIDCStore) ConsumeOIDCLoginTicket(_ context.Context, hash string, _ time.Time) (int64, error) {
	id, ok := s.tickets[hash]
	if !ok {
		return 0, ErrOIDCTicket
	}
	delete(s.tickets, hash)
	return id, nil
}

type fakeOIDCExchange struct {
	token                string
	err                  error
	gotCode, gotVerifier string
}

func (e *fakeOIDCExchange) Exchange(_ context.Context, code, verifier string) (string, error) {
	e.gotCode, e.gotVerifier = code, verifier
	return e.token, e.err
}

type fakeOIDCVerifier struct {
	claims   telegramOIDCClaims
	err      error
	gotNonce string
}

func (v *fakeOIDCVerifier) Verify(_ context.Context, token, nonce string) (telegramOIDCClaims, error) {
	v.gotNonce = nonce
	if token == "" || v.claims.Nonce != nonce {
		return telegramOIDCClaims{}, ErrOIDCIdentity
	}
	return v.claims, v.err
}

type fakeTokenIssuer struct {
	token   string
	expires time.Time
	userID  int64
}

func (i *fakeTokenIssuer) CreateToken(_ context.Context, id int64) (string, time.Time, error) {
	i.userID = id
	return i.token, i.expires, nil
}

func TestTelegramOIDCLoginProvisionAndTicketExchange(t *testing.T) {
	store := newFakeOIDCStore()
	exchanger := &fakeOIDCExchange{token: "signed-id-token"}
	verifier := &fakeOIDCVerifier{claims: telegramOIDCClaims{Subject: "subject-123", Name: "Telegram Name", Username: "sample"}}
	issuer := &fakeTokenIssuer{token: "api-bearer", expires: time.Now().Add(time.Hour)}
	flow := &telegramOIDCFlow{issuer: issuer, config: TelegramOIDCConfig{ClientID: "client", ClientSecret: "secret", RedirectURI: "https://api.example/callback", ReturnURLs: []string{"https://app.example/login"}}, store: store, exchanger: exchanger, verifier: verifier}
	start, err := flow.start(context.Background(), flow.config.ReturnURLs[0])
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := url.Parse(start)
	if err != nil {
		t.Fatal(err)
	}
	state := redirect.Query().Get("state")
	if redirect.Query().Get("code_challenge_method") != "S256" || redirect.Query().Get("nonce") == "" {
		t.Fatal("authorization request omitted PKCE or nonce")
	}
	tx := store.txns[sha256Hex(state)]
	verifier.claims.Nonce = tx.Nonce
	ticket, returnURL, err := flow.complete(context.Background(), state, "auth-code")
	if err != nil {
		t.Fatal(err)
	}
	if returnURL != flow.config.ReturnURLs[0] || ticket == "" || exchanger.gotCode != "auth-code" || exchanger.gotVerifier != tx.Verifier {
		t.Fatal("callback did not complete expected transaction")
	}
	if store.provisions != 1 {
		t.Fatalf("provisions=%d, want 1", store.provisions)
	}
	token, _, err := flow.exchangeTicket(context.Background(), ticket)
	if err != nil || token != "api-bearer" || issuer.userID != 1 {
		t.Fatalf("ticket exchange: token=%q user=%d err=%v", token, issuer.userID, err)
	}
	if _, _, err := flow.exchangeTicket(context.Background(), ticket); !errors.Is(err, ErrOIDCTicket) {
		t.Fatalf("reused ticket error=%v, want ErrOIDCTicket", err)
	}
}

func TestTelegramOIDCHandlersBindBrowserStateAndRedirectWithTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newFakeOIDCStore()
	verifier := &fakeOIDCVerifier{claims: telegramOIDCClaims{Subject: "handler-sub"}}
	issuer := &fakeTokenIssuer{token: "api-session-token", expires: time.Now().Add(time.Hour)}
	flow := &telegramOIDCFlow{
		issuer: issuer,
		config: TelegramOIDCConfig{ClientID: "client", ClientSecret: "secret", RedirectURI: "https://api.example/callback", ReturnURLs: []string{"https://app.example/login"}},
		store:  store, exchanger: &fakeOIDCExchange{token: "id-token"}, verifier: verifier,
	}
	handler := &TelegramOIDCHandler{flow: flow}

	startRecorder := httptest.NewRecorder()
	handler.Start(ginContext(startRecorder, http.MethodGet, "/?return_url=https%3A%2F%2Fapp.example%2Flogin", nil))
	if startRecorder.Code != http.StatusFound {
		t.Fatalf("start status=%d", startRecorder.Code)
	}
	startURL, err := url.Parse(startRecorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := startURL.Query().Get("state")
	stateCookie := startRecorder.Result().Cookies()[0]
	if !stateCookie.HttpOnly || !stateCookie.Secure || stateCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("state cookie is not protected: %+v", stateCookie)
	}
	verifier.claims.Nonce = store.txns[sha256Hex(state)].Nonce

	wrongBrowserRecorder := httptest.NewRecorder()
	wrongBrowser := ginContext(wrongBrowserRecorder, http.MethodGet, "/?state="+url.QueryEscape(state)+"&code=code", nil)
	handler.Callback(wrongBrowser)
	if wrongBrowserRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("callback without state cookie status=%d", wrongBrowserRecorder.Code)
	}
	if _, exists := store.txns[sha256Hex(state)]; !exists {
		t.Fatal("callback without browser cookie consumed the valid transaction")
	}

	callbackRecorder := httptest.NewRecorder()
	callback := ginContext(callbackRecorder, http.MethodGet, "/?state="+url.QueryEscape(state)+"&code=code", nil)
	callback.Request.AddCookie(stateCookie)
	handler.Callback(callback)
	if callbackRecorder.Code != http.StatusSeeOther {
		t.Fatalf("callback status=%d body=%s", callbackRecorder.Code, callbackRecorder.Body.String())
	}
	location, err := url.Parse(callbackRecorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(location.Fragment, "telegram_oidc_ticket=") || strings.Contains(location.String(), "api-session-token") {
		t.Fatalf("unsafe callback redirect: %s", location)
	}
	ticket, err := url.QueryUnescape(strings.TrimPrefix(location.Fragment, "telegram_oidc_ticket="))
	if err != nil || ticket == "" {
		t.Fatalf("callback did not include login ticket: %s", location.Fragment)
	}

	exchangeRecorder := httptest.NewRecorder()
	exchange := ginContext(exchangeRecorder, http.MethodPost, "/", bytes.NewBufferString(fmt.Sprintf(`{"ticket":%q}`, ticket)))
	handler.Exchange(exchange)
	if exchangeRecorder.Code != http.StatusOK || !strings.Contains(exchangeRecorder.Body.String(), `"access_token":"api-session-token"`) {
		t.Fatalf("exchange status=%d body=%s", exchangeRecorder.Code, exchangeRecorder.Body.String())
	}
}

func ginContext(recorder *httptest.ResponseRecorder, method, target string, body io.Reader) *gin.Context {
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, body)
	return c
}

func TestTelegramOIDCExistingIdentityAndConflict(t *testing.T) {
	store := newFakeOIDCStore()
	store.users["known-sub"] = User{ID: 47, Name: "Existing"}
	exchanger := &fakeOIDCExchange{token: "token"}
	verifier := &fakeOIDCVerifier{claims: telegramOIDCClaims{Subject: "known-sub"}}
	flow := &telegramOIDCFlow{issuer: &fakeTokenIssuer{}, config: TelegramOIDCConfig{ClientID: "client", ReturnURLs: []string{"https://app.example"}}, store: store, exchanger: exchanger, verifier: verifier}
	start, _ := flow.start(context.Background(), flow.config.ReturnURLs[0])
	u, _ := url.Parse(start)
	state := u.Query().Get("state")
	verifier.claims.Nonce = store.txns[sha256Hex(state)].Nonce
	if _, _, err := flow.complete(context.Background(), state, "code"); err != nil {
		t.Fatal(err)
	}
	if store.provisions != 0 {
		t.Fatal("existing OIDC subject was provisioned again")
	}

	store = newFakeOIDCStore()
	store.provisionErr = ErrOIDCIdentityConflict
	flow.store = store
	start, _ = flow.start(context.Background(), flow.config.ReturnURLs[0])
	u, _ = url.Parse(start)
	state = u.Query().Get("state")
	verifier.claims.Nonce = store.txns[sha256Hex(state)].Nonce
	if _, _, err := flow.complete(context.Background(), state, "code"); !errors.Is(err, ErrOIDCIdentityConflict) {
		t.Fatalf("identity conflict error=%v", err)
	}
}

func TestTelegramOIDCRejectsWrongStateNonceExpiredAndReusedCallback(t *testing.T) {
	store := newFakeOIDCStore()
	flow := &telegramOIDCFlow{config: TelegramOIDCConfig{ClientID: "client", ReturnURLs: []string{"https://app.example"}}, store: store, exchanger: &fakeOIDCExchange{token: "id-token"}, verifier: &fakeOIDCVerifier{claims: telegramOIDCClaims{Subject: "sub"}}}
	start, _ := flow.start(context.Background(), flow.config.ReturnURLs[0])
	u, _ := url.Parse(start)
	state := u.Query().Get("state")
	if _, _, err := flow.complete(context.Background(), "wrong-state", "code"); !errors.Is(err, ErrOIDCTransaction) {
		t.Fatalf("wrong state error=%v", err)
	}
	store.txns[sha256Hex(state)] = oidcTransaction{Nonce: "expected", Verifier: "verifier", ReturnURL: flow.config.ReturnURLs[0], ExpiresAt: time.Now().Add(-time.Second)}
	if _, _, err := flow.complete(context.Background(), state, "code"); !errors.Is(err, ErrOIDCTransaction) {
		t.Fatalf("expired state error=%v", err)
	}

	start, _ = flow.start(context.Background(), flow.config.ReturnURLs[0])
	u, _ = url.Parse(start)
	state = u.Query().Get("state")
	if _, _, err := flow.complete(context.Background(), state, "code"); err == nil {
		t.Fatal("wrong nonce was accepted")
	}
	if _, _, err := flow.complete(context.Background(), state, "code"); !errors.Is(err, ErrOIDCTransaction) {
		t.Fatalf("reused callback error=%v", err)
	}
}

func TestTelegramOIDCCodeExchangeFailure(t *testing.T) {
	store := newFakeOIDCStore()
	flow := &telegramOIDCFlow{config: TelegramOIDCConfig{ClientID: "client", ReturnURLs: []string{"https://app.example"}}, store: store, exchanger: &fakeOIDCExchange{err: errors.New("upstream unavailable")}, verifier: &fakeOIDCVerifier{}}
	start, _ := flow.start(context.Background(), flow.config.ReturnURLs[0])
	u, _ := url.Parse(start)
	if _, _, err := flow.complete(context.Background(), u.Query().Get("state"), "code"); err == nil {
		t.Fatal("code exchange failure was ignored")
	}
}

func TestTelegramOIDCJWKSVerifiesSignatureAndClaims(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	jwks := `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"key-a","n":"` + encode(privateKey.N.Bytes()) + `","e":"AQAB"}]}`
	verifier := newTelegramIDTokenVerifier("client-id")
	verifier.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != telegramOIDCJWKSURL {
			t.Fatalf("unexpected JWKS URL %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(jwks)), Header: make(http.Header)}, nil
	})
	claims := map[string]any{"iss": "https://oauth.telegram.org", "aud": "client-id", "sub": "stable-subject", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(), "nonce": "nonce-1"}
	compact := signTestJWT(t, privateKey, claims)
	got, err := verifier.Verify(context.Background(), compact, "nonce-1")
	if err != nil || got.Subject != "stable-subject" {
		t.Fatalf("valid token rejected: %+v %v", got, err)
	}
	parts := strings.Split(compact, ".")
	badSig := parts[0] + "." + parts[1] + "." + encode([]byte("invalid"))
	if _, err := verifier.Verify(context.Background(), badSig, "nonce-1"); !errors.Is(err, ErrOIDCIdentity) || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("invalid signature error=%v", err)
	}
	claims["aud"] = "another-client"
	if _, err := verifier.Verify(context.Background(), signTestJWT(t, privateKey, claims), "nonce-1"); !errors.Is(err, ErrOIDCIdentity) || !strings.Contains(err.Error(), "audience mismatch") {
		t.Fatalf("invalid audience error=%v", err)
	}
	claims["aud"], claims["exp"] = "client-id", time.Now().Add(-2*time.Minute).Unix()
	if _, err := verifier.Verify(context.Background(), signTestJWT(t, privateKey, claims), "nonce-1"); !errors.Is(err, ErrOIDCIdentity) || !strings.Contains(err.Error(), "ID token has expired") {
		t.Fatalf("expired token error=%v", err)
	}
	claims["aud"] = "client-id"
	claims["iat"] = time.Now().Add(-2 * time.Minute).Unix()
	claims["exp"] = time.Now().Add(-30 * time.Second).Unix()
	if _, err := verifier.Verify(context.Background(), signTestJWT(t, privateKey, claims), "nonce-1"); err != nil {
		t.Fatalf("token within allowed clock skew rejected: %v", err)
	}
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(time.Minute).Unix()
	if _, err := verifier.Verify(context.Background(), signTestJWT(t, privateKey, claims), "wrong-nonce"); !errors.Is(err, ErrOIDCIdentity) || !strings.Contains(err.Error(), "nonce mismatch") {
		t.Fatalf("invalid nonce error=%v", err)
	}
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	encode := base64.RawURLEncoding.EncodeToString
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "key-a", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	input := encode(header) + "." + encode(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + encode(signature)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
