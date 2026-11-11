# Oksana VPN API

## Go package layout

| Path | Responsibility |
| --- | --- |
| `cmd/api` | Start the executable. |
| `internal/app` | Read configuration and wire dependencies. |
| `internal/httpapi` | Configure Gin, CORS, health checks, and routes. |
| `internal/auth` | Password and Telegram login, tokens, and authentication middleware. |
| `internal/account` | Load and return the authenticated user's `/me` profile. |
| `internal/referral` | Referral rules, summary, and persistence. |
| `internal/telegram` | Parse and verify Telegram init data. |
| `internal/configs/vless` | Authenticated VLESS deep-link endpoint for the Mini App. |
| `internal/database` | MySQL connection and the existing SQL builder. |

Organize new code by feature. A feature package may contain several models and files; it does not need a package per table or endpoint. Keep HTTP request and response types near their handlers, and keep database row types in repositories. Pass only the facts another feature needs. For example, `referral.Participant` carries the user fields required by referral rules without importing the auth or account model.

## Telegram authentication

`POST /api/v1/auth/telegram` remains the Telegram Mini App login and accepts its existing `init_data` request. Browser login uses the separate OIDC Authorization Code flow:

1. Open `GET /api/v1/auth/telegram/oidc/start`. If several frontend return URLs are configured, pass the exact allowed URL as `return_url`.
2. Telegram returns to `GET /api/v1/auth/telegram/oidc/callback`. The API validates state, PKCE, and Telegram's ID token before provisioning or finding the account.
3. The callback redirects to the approved frontend URL with a short-lived, single-use `telegram_oidc_ticket` in the URL fragment. The SPA reads and removes the fragment, then posts `{"ticket":"..."}` to `POST /api/v1/auth/telegram/oidc/exchange`.
4. Ticket exchange returns the normal `access_token`, `token_type`, and `expires_at` JSON response. The access token is never put in a URL.

OIDC identities are stored by the verified Telegram OIDC `sub` claim. The signed `id` profile claim is also stored as `telegram_id` and may match an existing user only by exact Telegram ID; duplicate matches are treated as conflicts. Names and usernames are display fields and do not link accounts. An unmatched identity provisions a new user with its Telegram profile fields. If an OIDC-only account was provisioned before an exact Telegram ID match was available, a later verified login can reconcile its OIDC identity to the uniquely matching existing user.

Configure `TELEGRAM_OIDC_CLIENT_ID`, `TELEGRAM_OIDC_CLIENT_SECRET`, `TELEGRAM_OIDC_REDIRECT_URI`, and `TELEGRAM_OIDC_FRONTEND_RETURN_URLS` in the API environment. Return URLs are exact-match allowlisted HTTPS URLs; the callback URI must also be HTTPS. If these values are incomplete, the OIDC endpoints return `503` and do not start login. Apply the OIDC migration before enabling the flow. Telegram's `@BotFather` Login Widget settings must allow the SPA origin and the exact API callback URI, and its issued client ID and secret must be configured only on the backend.

The SPA must implement the ticket-fragment exchange to establish its bearer-token session. This backend change does not alter the Vue application.

`GET /api/v1/vless/link` returns only `happ_deep_link`, `incy_deep_link`, `v2raytun_deep_link`, and `show_raw_link`. The last field is always `false`. The three links use the authenticated user's UUID and `VLESS_PUBLIC_DOMAIN` (default `https://connect.oksana1984.ru`).

Authenticated users can set a standalone login and password with `PUT /api/v1/auth/credentials` and a bearer access token:

```json
{"login":"my-login","password":"at-least-8-characters"}
```

Logins are normalized to lowercase and allow 3-64 letters, digits, dots, underscores, or hyphens. Passwords are stored as bcrypt hashes. Subsequent password login uses `POST /api/v1/auth/login` with `{"login":"my-login","password":"..."}`. The existing `telegram` login field remains supported for accounts that predate standalone logins. A global `API_KEY` is not needed for user authentication; the issued bearer token is the per-user credential.

`internal/app` constructs repositories and services. `internal/httpapi` registers routes. Feature services own business decisions and repositories own SQL. Dependencies should flow from app and HTTP code toward features and database code, without feature import cycles.

## Checks

```sh
go test ./...
go vet ./...
```
