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
| `internal/database` | MySQL connection and the existing SQL builder. |

Organize new code by feature. A feature package may contain several models and files; it does not need a package per table or endpoint. Keep HTTP request and response types near their handlers, and keep database row types in repositories. Pass only the facts another feature needs. For example, `referral.Participant` carries the user fields required by referral rules without importing the auth or account model.

`internal/app` constructs repositories and services. `internal/httpapi` registers routes. Feature services own business decisions and repositories own SQL. Dependencies should flow from app and HTTP code toward features and database code, without feature import cycles.

## Checks

```sh
go test ./...
go vet ./...
```
