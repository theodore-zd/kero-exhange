# CLAUDE.md - Project Context for kero-exchange

## What is this project?

kero-exchange is a centralized currency exchange server built in Go. It provides a REST API and web admin interface for managing multi-currency wallets, balances, and transactions. Providers (external services) can generate reference codes for user signup, and admins manage the full system via a dedicated UI.

## Quick Reference

```bash
# Build
./scripts/build-server.sh          # Compiles to tmp/kero-server

# Dev (migrate + build + run on port 8090)
./scripts/dev.sh [port]

# Migrations (requires goose CLI)
./scripts/migrate.sh up|down|status

# PostgreSQL setup (interactive Docker setup)
./scripts/deploy-postgres.sh

# Tests
go test ./...                       # All tests
go test -v ./tests -run TestName    # Single test
go test -race ./...                 # Race detection
go test -cover ./...                # With coverage

# Code quality
go fmt ./... && go vet ./...
go mod tidy
```

## Architecture

```
cmd/server/main.go        Entry point - loads config, connects DB, starts HTTP server
internal/
  config/                  Env-based config (postgres.env, exchange.env)
  crypto/                  Passphrase hashing (bcrypt via x/crypto)
  db/                      Database layer - pgx/pgxpool, models, queries
  handlers/                HTTP handlers (API JSON + admin/user web templates)
  middleware/              Auth (API key, access token, admin session), rate limiting
    context/               Context key definitions for middleware
  services/                Business logic between handlers and DB
migrations/                Goose SQL migrations (PostgreSQL)
templates/                 Grove templates (.grov files)
  base.grov                Root layout (title, head, nav, content, scripts blocks)
  admin/                   Admin panel pages (extends admin/base.grov)
  user/                    User-facing pages (extends user/base.grov)
  components/              Reusable components (alert, nav, pagination)
static/                    Plain CSS and JS (no frameworks)
tests/                     All tests live here (not alongside source)
scripts/                   Bash build/deploy/migrate scripts
```

### Request flow

```
HTTP Request → chi router → middleware (auth, rate limit) → handler → service → db → PostgreSQL
```

### Key dependencies

- **Router:** go-chi/chi/v5
- **Database:** jackc/pgx/v5 (connection pooling via pgxpool)
- **Migrations:** pressly/goose/v3
- **Precision:** shopspring/decimal (DECIMAL(20,8) for financial amounts)
- **Auth:** golang.org/x/crypto (bcrypt for passphrases, tokens, API keys)
- **Templating:** wispberry-tech/grove (Jinja2-like template engine)
- **Logging:** wispberry-tech/go-common (structured logging)
- **UUIDs:** google/uuid (all primary keys)
- **Env loading:** joho/godotenv

## Authentication Model

Three auth mechanisms, each as chi middleware:

1. **API Key auth** - Providers send `X-API-Key` header. Hash compared against `providers` table. Used for reference code generation and user signup.
2. **Access Token auth** - Users send `Authorization: Bearer <token>`. Hash compared against `access_tokens` table. Used for all user-facing API endpoints.
3. **Admin session auth** - Cookie-based session after login with admin password. Used for admin API and web routes.

## Database

- **Engine:** PostgreSQL 14+
- **Driver:** pgx/v5 with pgxpool
- **Primary keys:** UUID everywhere
- **Financial amounts:** DECIMAL(20,8) via shopspring/decimal
- **Soft deletes:** `deleted_at` column on wallets, currencies, balances
- **Pagination:** `db.PaginationParams` with `Normalize()` (default 20, max 100)
- **Not-found convention:** DB query functions return `nil, nil` (not an error)

### Core models

| Model | Purpose |
|-------|---------|
| Wallet | User wallet with passphrase_hash, access_token_hash |
| Currency | Currency definition (code, name, description) |
| Balance | Wallet-currency pair, DECIMAL(20,8) amount |
| Transaction | Ledger entry (deposit, withdrawal, transfer, admin_issued); transfer_id links paired debit/credit |
| Provider | External API provider with hashed API key |
| ReferenceCode | Single-use signup codes (1hr expiry) |
| AccessToken | Bearer tokens for user API access |
| AuditLog | Admin action audit trail |

## API Structure

Base path: `/api/v1`

- **Public:** `GET /health`, `POST /api/v1/admin/login`, `POST /api/v1/auth/signin`
- **API Key protected:** `POST /api/v1/providers/reference-codes`, `POST /api/v1/auth/signup`
- **Access Token protected:** `GET` on `/api/v1/wallets`, `/api/v1/currencies`, `/api/v1/balances`, `/api/v1/transactions`, `GET /api/v1/dashboard`, `POST /api/v1/transfers`
- **Admin protected:** Full management under `/api/v1/admin/` (providers, wallets, currencies, transactions, audit-logs)
- **User web:** `GET /signin`, `GET /dashboard` (Grove-rendered, JS-driven)
- **Admin web:** Server-rendered pages under `/admin/` (login, dashboard, CRUD forms, lookup)

JSON responses use `common.WriteJSONResponse` / `common.WriteJSONError`. Paginated endpoints return `{data: [...], meta: {page, page_size, total, total_pages}}`.

## Code Conventions

### Patterns to follow

- **Handlers are thin** - parse request, call service, write response. No business logic in handlers.
- **Response DTOs** - never expose internal models directly; use separate response types that hide sensitive fields.
- **Error wrapping** - use `fmt.Errorf("context: %w", err)` for all error propagation.
- **Context propagation** - `context.Context` as first param on all service/db methods.
- **Constructor pattern** - `NewXxxService(pool)`, `NewXxxHandler(svc)`.
- **Receiver names** - single lowercase letter matching type (`s` for `*WalletService`).
- **Import order** - stdlib, then third-party, then internal packages, separated by blank lines.
- **Timestamps** - format as RFC3339: `.UTC().Format("2006-01-02T15:04:05Z")`.
- **JSON tags** - snake_case field names, `omitempty` for optional, pointer types for nullable.

### Testing

- All tests live in `tests/` package (separate from source).
- `TestMain` handles DB setup/teardown.
- Test helpers: `createTestWallet`, `createTestCurrency`, etc. with matching `deleteTestXxx` cleanup.
- `setupTestServer` creates httptest server instances.
- Test DB: `postgresql://postgres:postgres@localhost:5443/local_pg`
- Run specific tests: `go test -v ./tests -run TestFunctionName`

### Templates & Static Assets

- **Grove** template engine (`wispberry-tech/grove`) with Jinja2-like syntax (`.grov` files).
- Block inheritance: `{% extends "path" %}`, `{% block name %}...{% endblock %}`.
- Component inclusion: `{% component "path" %}...{% endcomponent %}`, `{% props var1, var2 %}`.
- Variable interpolation: `{{ variable }}`, conditionals: `{% if condition %}...{% endif %}`.
- Layout hierarchy: `base.grov` → `admin/base.grov` or `user/base.grov` → page templates.
- Plain CSS in `static/css/main.css`, plain JS in `static/js/` (no build tools or frameworks).
- JS files: `app.js` (user auth/dashboard), `admin.js` (admin panel), `utils.js` (shared helpers).

## Environment Configuration

Two env files required (see `*.env.example` for templates):

**postgres.env** - Database connection:
- `DATABASE_URL` (required) - PostgreSQL connection string
- `GOOSE_DRIVER=postgres`, `GOOSE_DBSTRING` - for migrations

**exchange.env** - Application config:
- `ADMIN_PASSWORD` or `ADMIN_PASSWORD_HASH` (one required) - Admin login credentials
- `PORT` (default: 8080), `LOG_LEVEL` (default: info)
- `DEFAULT_CURRENCY_CODE` (default: USD), `DEFAULT_CURRENCY_NAME`, `DEFAULT_CURRENCY_DESCRIPTION`
- `SEED_MODE` (optional, `"true"` to enable) - Seeds test wallets, provider, and balances on startup

## Things to Know

- The binary output goes to `tmp/kero-server` - this directory is gitignored.
- `_local_deploy/` contains Docker PostgreSQL data - also gitignored, never commit.
- Rate limiting is set to 120 requests/minute globally.
- The `go-common` package (`wispberry-tech/go-common`) provides logging utilities used everywhere.
- `common.WriteJSONResponse` wraps the payload in a `{"data": ...}` envelope. In JS, name the parsed response variable `result` (not `data`) so field access reads `result.data.field` — never `data.data.field`.
