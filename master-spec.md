# kero-exchange Master Technical Specification

> **Living document** -- keep updated as the system evolves. Last updated: 2026-04-08.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Architecture](#2-architecture)
3. [Startup Sequence](#3-startup-sequence)
4. [Configuration](#4-configuration)
5. [Database Layer](#5-database-layer)
6. [Authentication & Authorization](#6-authentication--authorization)
7. [API Reference](#7-api-reference)
8. [Service Layer](#8-service-layer)
9. [Transfer System](#9-transfer-system)
10. [Template Engine (Grove)](#10-template-engine-grove)
11. [Frontend Architecture](#11-frontend-architecture)
12. [Middleware Stack](#12-middleware-stack)
13. [Migration History](#13-migration-history)
14. [Seed System](#14-seed-system)
15. [Error Handling Conventions](#15-error-handling-conventions)
16. [Testing](#16-testing)
17. [Build & Deployment](#17-build--deployment)
18. [Dependencies](#18-dependencies)

---

## 1. System Overview

kero-exchange is a centralized, multi-currency exchange server written in Go. It exposes a RESTful JSON API and a server-rendered web interface for both end users and administrators.

**Core capabilities:**

- Multi-currency wallet management with DECIMAL(20,8) precision
- Peer-to-peer transfers between wallets (atomic debit/credit pairs)
- Provider-based reference code system for user onboarding
- Admin panel for full system management with audit logging
- User dashboard for balance viewing and transfers

**Design principles:**

- Thin handlers, thick services, direct SQL queries (no ORM)
- UUID primary keys everywhere, soft deletes where appropriate
- All financial amounts use `shopspring/decimal` -- never float
- Template-driven server rendering with client-side JS for auth flows

---

## 2. Architecture

### Directory Structure

```
kero-exchange/
  cmd/server/main.go             Entry point
  internal/
    config/config.go              Environment-based configuration
    crypto/                       Passphrase hashing (bcrypt via x/crypto)
    db/                           Database models and query functions
      db.go                       Connection pool, pagination helpers (PaginationParams, Paginate[T])
      wallet.go                   Wallet CRUD, search
      currency.go                 Currency CRUD
      balance.go                  Balance queries, summary, FOR UPDATE locking
      transaction.go              Transaction CRUD, filtering, transfer_id linking
      provider.go                 Provider CRUD
      reference_code.go           Reference code lifecycle (create, lookup, mark used)
      access_token.go             Access token CRUD, expiry cleanup
      audit_log.go                Audit log creation and filtered listing
    handlers/
      routes.go                   Chi router setup, route registration, middleware wiring
      render.go                   Grove template engine initialization and rendering
      helpers.go                  UUID parsing, error response helpers
      common.go                   PaginationMeta struct
      auth.go                     SignIn, SignUp, GenerateReferenceCode
      wallet.go                   Wallet list/get
      currency.go                 Currency list/get/getByCode
      balance.go                  Balance list/get
      transaction.go              Transaction list/get
      transfer.go                 Transfer creation
      dashboard.go                Dashboard summary
      admin_api.go                Admin JSON API (CRUD for all entities)
      admin_types.go              Admin request/response DTOs
      admin_web.go                Admin Grove page handlers
      web.go                      User Grove page handlers (signin, dashboard)
    middleware/
      auth.go                     APIKeyMiddleware, AccessTokenMiddleware
      admin.go                    AdminAuthMiddleware, in-memory token store
      ratelimit.go                Token bucket rate limiter
      context/                    Context key definitions (ProviderUUID, WalletUUID)
    services/
      auth.go                     Signup/signin business logic
      wallet.go                   Wallet operations
      currency.go                 Currency operations, EnsureDefaultCurrency
      balance.go                  Balance queries
      transaction.go              Transaction queries
      transfer.go                 Atomic wallet-to-wallet transfers
      dashboard.go                Dashboard aggregation (wallet count + balance summary)
      admin.go                    Admin login, provider/wallet/currency management
      audit_log.go                Audit log recording
      seed.go                     Development data seeding
  migrations/                     Goose SQL migrations (12 files)
  templates/                      Grove template files (.grov)
  static/
    css/main.css                  Application styles
    js/app.js                     User-side auth and dashboard logic
    js/admin.js                   Admin panel client logic
    js/utils.js                   Shared JS utilities
  tests/                          Integration tests (separate package)
  scripts/                        Build, dev, migrate, deploy scripts
```

### Request Flow

```
Client Request
  |
  v
chi.Router
  |
  +-- Global middleware: Logger, Recoverer, RequestID, RateLimiter(120/min)
  |
  +-- Route group middleware (one of):
  |     APIKeyMiddleware       -> sets context.ProviderUUID
  |     AccessTokenMiddleware  -> sets context.WalletUUID
  |     AdminAuthMiddleware    -> validates in-memory session token
  |
  v
Handler (parse request, call service, write response)
  |
  v
Service (business logic, validation, orchestration)
  |
  v
DB query functions (direct SQL via pgx, connection pool)
  |
  v
PostgreSQL
```

---

## 3. Startup Sequence

Defined in `cmd/server/main.go`:

1. **Initialize logger** -- `common.InitializeLogger()`
2. **Load config** -- reads `postgres.env` + `exchange.env`, validates required fields
3. **Set log level** -- from `LOG_LEVEL` env var
4. **Connect database** -- `db.NewPool()` with `DATABASE_URL`
5. **Ensure default currency** -- creates if not exists (USD by default)
6. **Seed data** (conditional) -- if `SEED_MODE=true`, runs `SeedService.Seed()`
7. **Create chi router** -- `handlers.RegisterRoutes()` wires all routes + middleware
8. **Start HTTP server** -- with timeouts: read 15s, write 15s, idle 60s
9. **Graceful shutdown** -- listens for SIGINT/SIGTERM, 30s shutdown timeout

---

## 4. Configuration

**Source:** `internal/config/config.go`

```go
type Config struct {
    DatabaseURL                string   // Required. PostgreSQL connection string
    Port                       string   // Default: "8080"
    LogLevel                   string   // Default: "info"
    AdminPassword              string   // Plaintext admin password (one of these required)
    AdminPasswordHash          string   // Pre-hashed admin password (one of these required)
    DefaultCurrencyCode        string   // Default: "USD"
    DefaultCurrencyName        string   // Default: "US Dollar"
    DefaultCurrencyDescription string   // Optional
    SeedMode                   bool     // Set via SEED_MODE="true"
}
```

**Env files loaded (via godotenv):**

| File | Variables |
|------|-----------|
| `postgres.env` | `DATABASE_URL`, `GOOSE_DRIVER`, `GOOSE_DBSTRING` |
| `exchange.env` | `ADMIN_PASSWORD` or `ADMIN_PASSWORD_HASH`, `PORT`, `LOG_LEVEL`, `DEFAULT_CURRENCY_CODE`, `DEFAULT_CURRENCY_NAME`, `DEFAULT_CURRENCY_DESCRIPTION`, `SEED_MODE` |

**Validation rules:**
- `DATABASE_URL` must be non-empty
- At least one of `ADMIN_PASSWORD` or `ADMIN_PASSWORD_HASH` must be set

---

## 5. Database Layer

### Connection

- **Engine:** PostgreSQL 14+
- **Driver:** `jackc/pgx/v5` with `pgxpool` connection pooling
- **Pool creation:** `db.NewPool(ctx, connString)` returns `*pgxpool.Pool`

### Conventions

- **Primary keys:** `UUID` type everywhere (generated by PostgreSQL `gen_random_uuid()`)
- **Financial precision:** `DECIMAL(20,8)` stored as `shopspring/decimal.Decimal` in Go
- **Soft deletes:** `deleted_at TIMESTAMPTZ` column on wallets, currencies, balances, transactions. Queries filter `WHERE deleted_at IS NULL`
- **Not-found convention:** Query functions return `(nil, nil)` when no row matches -- not an error
- **Timestamps:** All `TIMESTAMPTZ`, formatted as RFC3339 in JSON (`2006-01-02T15:04:05Z`)

### Pagination

```go
type PaginationParams struct {
    Page     int   // 1-based, default 1
    PageSize int   // default 20, max 100
}

type PaginatedResult[T any] struct {
    Data       []T
    Total      int64
    Page       int
    PageSize   int
    TotalPages int
}
```

The `Paginate[T]()` generic function handles count query + data query + offset/limit calculation.

### Data Models

#### Wallet

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| PassphraseHash | `string` | bcrypt hash of user passphrase |
| AccessTokenHash | `string` | Hash stored on wallet for legacy lookup |
| CreatedAt | `time.Time` | |
| UpdatedAt | `time.Time` | Updated on passphrase/token changes |

**Key operations:** Create, Get by UUID, Get by access token hash, Get by passphrase hash, Update passphrase hash, Update access token hash, Delete (hard), Search by UUID prefix.

#### Currency

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| Code | `string` | Unique currency code (e.g., "USD") |
| Name | `string` | Display name |
| Description | `*string` | Optional |
| CreatedAt | `time.Time` | |
| DeletedAt | `*time.Time` | Soft delete |

**Key operations:** Create, Get by UUID, Get by code, List (paginated, ordered by code ASC), Update (name + description), Delete (soft -- cascades to balances and transactions).

**Cascade on delete:** Soft-deleting a currency also soft-deletes all associated balances and transactions in a single DB transaction.

#### Balance

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| WalletID | `uuid.UUID` | FK to wallet |
| CurrencyID | `uuid.UUID` | FK to currency |
| Balance | `decimal.Decimal` | DECIMAL(20,8), non-negative (enforced by DB check constraint `balances_balance_check`) |
| UpdatedAt | `time.Time` | |
| CurrencyCode | `string` | Joined from currencies table |
| CurrencyName | `string` | Joined from currencies table |
| DeletedAt | `*time.Time` | Soft delete |

**Key operations:** Get by UUID, Get by wallet+currency pair, Get with `FOR UPDATE` lock (used in transfers), List (paginated, filtered by wallet/currency), Balance summary by wallet (grouped by currency).

**Balance trigger:** A PostgreSQL trigger automatically updates the balance when transactions are inserted (see migration `20260312191500_add_balance_trigger.sql`, fixed in `20260408000000_fix_balance_trigger.sql`). This means inserting a transaction automatically adjusts the corresponding balance row.

#### Transaction

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| WalletID | `uuid.UUID` | FK to wallet |
| CurrencyID | `uuid.UUID` | FK to currency |
| Amount | `decimal.Decimal` | Positive for credits, negative for debits |
| Type | `TransactionType` | `"deposit"`, `"withdrawal"`, `"transfer"`, `"admin_issued"` |
| Reference | `*string` | Optional reference string |
| TransferID | `*uuid.UUID` | Links debit+credit pair in transfers; NULL for other types |
| Timestamp | `time.Time` | Creation time |
| DeletedAt | `*time.Time` | Soft delete |

**Transaction types:**
- `deposit` -- funds added to wallet
- `withdrawal` -- funds removed from wallet
- `transfer` -- peer-to-peer; always created in pairs sharing the same `transfer_id`
- `admin_issued` -- funds issued by admin to a wallet

**Key operations:** Create (pool or tx variant), Get by UUID, List (paginated, filtered by wallet/currency/type/date range, ordered by timestamp DESC), Get by transfer ID.

#### Provider

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| APIKeyHash | `string` | SHA-256 hash of the API key |
| Name | `string` | Provider display name |
| CreatedAt | `time.Time` | |

**Key operations:** Create, Get by API key hash, List (paginated), Update API key hash, Delete (hard).

#### ReferenceCode

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| Code | `string` | The reference code string |
| ProviderID | `uuid.UUID` | FK to provider that created it |
| UsedAt | `*time.Time` | NULL until consumed by signup |
| ExpiresAt | `time.Time` | 1-hour expiry from creation |
| CreatedAt | `time.Time` | |

**Key operations:** Create, Get by code (with optional `FOR UPDATE` lock), Mark used (sets `used_at = NOW()`), Delete (hard).

**Lifecycle:** Provider generates code via API -> User presents code during signup -> Code is validated (not expired, not used) -> Marked used atomically within signup transaction.

#### AccessToken

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| WalletID | `uuid.UUID` | FK to wallet |
| Token | `string` | The bearer token string |
| ExpiresAt | `time.Time` | Token expiration |
| CreatedAt | `time.Time` | |
| LastUsedAt | `*time.Time` | Updated on each authenticated request |

**Key operations:** Create, Get by token string, Update last used, Delete by wallet ID, Delete expired tokens.

#### AuditLog

| Field | Type | Notes |
|-------|------|-------|
| UUID | `uuid.UUID` | Primary key |
| Action | `string` | Action performed (e.g., "create_provider") |
| EntityType | `string` | Type of entity affected |
| EntityID | `*uuid.UUID` | ID of affected entity (nullable) |
| Details | `map[string]interface{}` | JSON details stored as JSONB |
| AdminUser | `string` | Identifier of admin who performed action |
| IPAddress | `string` | Request IP |
| UserAgent | `string` | Request user agent |
| CreatedAt | `time.Time` | |

**Key operations:** Create, List (paginated, filtered by action/entity type/entity ID/date range, ordered by created_at DESC).

---

## 6. Authentication & Authorization

Three independent auth mechanisms, implemented as chi middleware:

### 6.1 API Key Authentication

**Middleware:** `middleware.APIKeyMiddleware(pool)`
**Header:** `X-API-Key: <api-key>`
**Flow:**
1. Extract `X-API-Key` header
2. Hash the key with `crypto.HashPassphrase()` (SHA-256)
3. Look up provider by hash in `providers` table
4. If found, set `context.ProviderUUID` on request context
5. If missing/invalid, return 401 with `MISSING_API_KEY` or `INVALID_API_KEY`

**Used by:** Reference code generation, user signup

### 6.2 Access Token Authentication

**Middleware:** `middleware.AccessTokenMiddleware(pool)`
**Header:** `Authorization: Bearer <token>`
**Flow:**
1. Extract and parse `Authorization` header (must be `Bearer <token>`)
2. Look up token in `access_tokens` table by raw token string
3. Check expiration (`expires_at > NOW()`)
4. Update `last_used_at` timestamp
5. Set `context.WalletUUID` on request context (from `token.WalletID`)
6. If missing/invalid/expired, return 401

**Used by:** All user-facing API endpoints (wallets, currencies, balances, transactions, transfers, dashboard)

### 6.3 Admin Session Authentication

**Middleware:** `middleware.AdminAuthMiddleware`
**Header:** `Authorization: Bearer <admin-token>`
**Flow:**
1. Admin logs in via `POST /api/v1/admin/login` with password
2. Server generates a random token, stores it in an **in-memory** `AdminTokenStore` (sync.RWMutex-protected map) with an expiry duration
3. Subsequent requests include the token as `Authorization: Bearer <token>`
4. Middleware validates token exists in store and hasn't expired
5. Expired tokens are lazily cleaned up on validation

**Token store:** `middleware.AdminTokenStore` -- thread-safe in-memory map of `token -> expiry time`. Tokens do not survive server restarts.

**Functions:** `StoreAdminToken(token, expiry)`, `RevokeAdminToken(token)`, `adminTokenStore.ValidateToken(token)`

---

## 7. API Reference

### Route Groups

#### Public Routes (no auth)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | redirect | Redirects to `/signin` |
| GET | `/health` | `healthHandler` | Returns `{"status": "healthy"}` |
| GET | `/signin` | `WebHandler.SignInPage` | User sign-in page (Grove template) |
| GET | `/dashboard` | `WebHandler.DashboardPage` | User dashboard page (Grove template) |
| POST | `/api/v1/auth/signin` | `AuthHandler.SignIn` | Authenticate with passphrase, returns access token |
| POST | `/api/v1/admin/login` | `AdminAPIHandler.Login` | Admin login, returns session token |
| GET | `/admin/login` | `AdminWebHandler.LoginPage` | Admin login page |
| GET | `/admin` | redirect | Redirects to `/admin/dashboard` |
| GET | `/admin/dashboard` | `AdminWebHandler.DashboardPage` | Admin dashboard page |
| GET | `/admin/providers` | `AdminWebHandler.ProvidersPage` | Provider list page |
| GET | `/admin/providers/new` | `AdminWebHandler.ProviderCreatePage` | Create provider form |
| GET | `/admin/providers/{id}/edit-api-key` | `AdminWebHandler.ProviderEditPage` | Edit provider API key |
| GET | `/admin/wallets` | `AdminWebHandler.WalletsPage` | Wallet list page |
| GET | `/admin/wallets/new` | `AdminWebHandler.WalletCreatePage` | Create wallet form |
| GET | `/admin/wallets/{id}` | `AdminWebHandler.WalletDetailPage` | Wallet detail view |
| GET | `/admin/wallets/{id}/regenerate` | `AdminWebHandler.WalletRegeneratePage` | Regenerate passphrase form |
| GET | `/admin/wallets/{id}/issue-currency` | `AdminWebHandler.WalletIssueCurrencyPage` | Issue currency form |
| GET | `/admin/currencies` | `AdminWebHandler.CurrenciesPage` | Currency list page |
| GET | `/admin/currencies/new` | `AdminWebHandler.CurrencyCreatePage` | Create currency form |
| GET | `/admin/currencies/{id}/edit` | `AdminWebHandler.CurrencyEditPage` | Edit currency form |
| GET | `/admin/transactions` | `AdminWebHandler.TransactionsPage` | Transaction list page |
| GET | `/admin/audit-log` | `AdminWebHandler.AuditLogPage` | Audit log viewer |
| GET | `/admin/lookup` | `AdminWebHandler.LookupPage` | Wallet lookup/search |
| GET | `/static/*` | `http.FileServer` | Static file serving |

> **Note:** Admin web pages are public (no server-side auth gate). The pages themselves call the admin API endpoints using JS with the admin token from localStorage. The admin API endpoints enforce auth.

#### API Key Protected Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/v1/providers/reference-codes` | `AuthHandler.GenerateReferenceCode` | Generate a single-use signup reference code |
| POST | `/api/v1/auth/signup` | `AuthHandler.SignUp` | Create wallet using reference code |

#### Access Token Protected Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/dashboard` | `DashboardHandler.Summary` | Wallet count + balance summary |
| GET | `/api/v1/wallets` | `WalletHandler.List` | List wallets (paginated) |
| GET | `/api/v1/wallets/{id}` | `WalletHandler.Get` | Get wallet by UUID |
| GET | `/api/v1/currencies` | `CurrencyHandler.List` | List currencies (paginated) |
| GET | `/api/v1/currencies/{id}` | `CurrencyHandler.Get` | Get currency by UUID |
| GET | `/api/v1/currencies/code/{code}` | `CurrencyHandler.GetByCode` | Get currency by code |
| GET | `/api/v1/balances` | `BalanceHandler.List` | List balances (paginated, filterable) |
| GET | `/api/v1/balances/{id}` | `BalanceHandler.Get` | Get balance by UUID |
| GET | `/api/v1/transactions` | `TransactionHandler.List` | List transactions (paginated, filterable) |
| GET | `/api/v1/transactions/{id}` | `TransactionHandler.Get` | Get transaction by UUID |
| POST | `/api/v1/transfers` | `TransferHandler.Create` | Transfer funds between wallets |

#### Admin Protected Routes (Bearer token from admin login)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/v1/admin/providers` | `AdminAPIHandler.CreateProvider` | Create provider, returns API key |
| GET | `/api/v1/admin/providers` | `AdminAPIHandler.ListProviders` | List providers (paginated) |
| PUT | `/api/v1/admin/providers/{id}` | `AdminAPIHandler.UpdateProvider` | Regenerate provider API key |
| DELETE | `/api/v1/admin/providers/{id}` | `AdminAPIHandler.DeleteProvider` | Delete provider |
| POST | `/api/v1/admin/wallets` | `AdminAPIHandler.CreateWallet` | Create wallet, returns passphrase + token |
| GET | `/api/v1/admin/wallets` | `AdminAPIHandler.ListWallets` | List wallets (paginated) |
| GET | `/api/v1/admin/wallets/search` | `AdminAPIHandler.SearchWallets` | Search wallets by UUID prefix |
| POST | `/api/v1/admin/wallets/{id}/regenerate` | `AdminAPIHandler.RegenerateWalletPassphrase` | Regenerate wallet passphrase |
| DELETE | `/api/v1/admin/wallets/{id}` | `AdminAPIHandler.DeleteWallet` | Delete wallet |
| POST | `/api/v1/admin/wallets/{id}/issue-currency` | `AdminAPIHandler.IssueCurrencyToWallet` | Issue currency balance to wallet |
| GET | `/api/v1/admin/wallets/{id}/balances` | `AdminAPIHandler.GetWalletBalances` | Get all balances for a wallet |
| POST | `/api/v1/admin/currencies` | `AdminAPIHandler.CreateCurrency` | Create currency |
| GET | `/api/v1/admin/currencies` | `AdminAPIHandler.ListCurrencies` | List currencies (paginated) |
| PUT | `/api/v1/admin/currencies/{id}` | `AdminAPIHandler.UpdateCurrency` | Update currency name/description |
| DELETE | `/api/v1/admin/currencies/{id}` | `AdminAPIHandler.DeleteCurrency` | Soft-delete currency (cascades) |
| GET | `/api/v1/admin/transactions` | `AdminAPIHandler.ListTransactions` | List all transactions (paginated) |
| GET | `/api/v1/admin/transactions/{id}` | `AdminAPIHandler.GetTransaction` | Get transaction by UUID |
| GET | `/api/v1/admin/audit-logs` | `AdminAPIHandler.ListAuditLogs` | List audit logs (paginated, filterable) |

### Response Format

**Success (single entity):**
```json
{
  "data": { ... }
}
```

**Success (paginated list):**
```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**Error:**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": { ... }
  }
}
```

Note: `common.WriteJSONResponse` wraps the payload in `{"data": ...}`. In JavaScript, name the parsed response variable `result` (not `data`) so field access reads `result.data.field` -- never `data.data.field`.

---

## 8. Service Layer

Services contain business logic and sit between handlers and the DB layer. Each service receives a `*pgxpool.Pool` at construction.

| Service | Constructor | Responsibilities |
|---------|-------------|------------------|
| `WalletService` | `NewWalletService(pool)` | Wallet CRUD |
| `CurrencyService` | `NewCurrencyService(pool)` | Currency CRUD, `EnsureDefaultCurrency()` |
| `BalanceService` | `NewBalanceService(pool)` | Balance queries |
| `TransactionService` | `NewTransactionService(pool)` | Transaction queries |
| `TransferService` | `NewTransferService(pool)` | Atomic peer-to-peer transfers |
| `AuthService` | `NewAuthService(pool)` | Signup (with reference code), signin (passphrase -> token) |
| `AdminService` | `NewAdminService(pool, pwd, hash)` | Admin login, provider/wallet/currency management, audit logging |
| `DashboardService` | `NewDashboardService(pool)` | Dashboard summary (wallet count + balance aggregation) |
| `AuditLogService` | `NewAuditLogService(pool)` | Audit log recording |
| `SeedService` | `NewSeedService(pool)` | Development data seeding |

**Pattern:** All service methods take `context.Context` as first parameter. Constructor receivers use single lowercase letter (e.g., `s` for service).

---

## 9. Transfer System

The transfer system enables atomic peer-to-peer fund movement between wallets.

### Flow

```
POST /api/v1/transfers
  Body: { destination_wallet_id, currency_id, amount }
  Auth: Access Token (source wallet derived from token)
```

1. **Validation:**
   - Source and destination must be different wallets (`ErrSameWallet`)
   - Amount must be positive (`ErrInsufficientBalance`)
   - Destination wallet must exist (`ErrDestinationNotFound`)
   - Currency must exist (`ErrCurrencyNotFound`)

2. **Database transaction** (serializable via `pgx.TxOptions{}`):
   - Acquire source balance row with `SELECT ... FOR UPDATE` (row-level lock prevents race conditions)
   - Verify source balance >= transfer amount (`ErrInsufficientBalance`)
   - Generate shared `transfer_id` UUID
   - Insert **debit** transaction: source wallet, negative amount, type `"transfer"`, transfer_id
   - Insert **credit** transaction: destination wallet, positive amount, type `"transfer"`, transfer_id
   - Commit transaction

3. **Balance update:** The PostgreSQL balance trigger automatically adjusts both wallets' balance rows when the transactions are inserted.

4. **Response (201):**
   ```json
   {
     "data": {
       "transfer_id": "uuid",
       "debit": { ... transaction details ... },
       "credit": { ... transaction details ... }
     }
   }
   ```

### Race Condition Protection

- `FOR UPDATE` lock on the source balance row prevents concurrent transfers from the same wallet from both succeeding when balance is insufficient
- Database check constraint `balances_balance_check` provides a secondary safety net (balance cannot go negative)
- If the constraint triggers, the service catches the error string `"balances_balance_check"` and returns `ErrInsufficientBalance`

### Defined Errors

```go
var (
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrSameWallet          = errors.New("source and destination wallet are the same")
    ErrDestinationNotFound = errors.New("destination wallet not found")
    ErrCurrencyNotFound    = errors.New("currency not found")
)
```

---

## 10. Template Engine (Grove)

The frontend uses **Grove** (`wispberry-tech/grove`), a Jinja2-like template engine for Go.

### Initialization

In `internal/handlers/render.go`:
- `InitGrove()` creates a `grove.FileSystemStore` pointing to the `templates/` directory
- Cache size: 256 templates
- `renderGrove(w, templatePath, data)` renders a template with a `grove.Data` map and sets `Content-Type: text/html; charset=utf-8`

### Template Syntax

| Feature | Syntax |
|---------|--------|
| Block inheritance | `{% extends "base.grov" %}` |
| Block definition | `{% block content %}...{% endblock %}` |
| Component include | `{% component "components/alert.grov" %}...{% endcomponent %}` |
| Props declaration | `{% props error, success %}` |
| Variable output | `{{ variable }}` |
| Conditionals | `{% if condition %}...{% endif %}` |
| Loops | `{% for item in items %}...{% endfor %}` |

### Template Hierarchy

```
templates/base.grov                    Root layout
  |-- Blocks: title, head, nav, content, scripts
  |
  +-- templates/admin/base.grov        Admin layout (extends base.grov)
  |     |-- Adds sidebar navigation
  |     |-- Block: content (within main area)
  |     |
  |     +-- admin/dashboard.grov       Stats cards
  |     +-- admin/login.grov           Login form
  |     +-- admin/providers.grov       Provider list
  |     +-- admin/provider-form.grov   Create/edit provider
  |     +-- admin/wallets.grov         Wallet list
  |     +-- admin/wallet-detail.grov   Single wallet view
  |     +-- admin/wallet-form.grov     Create/edit wallet
  |     +-- admin/currencies.grov      Currency list
  |     +-- admin/currency-form.grov   Create/edit currency
  |     +-- admin/transactions.grov    Transaction list
  |     +-- admin/audit-log.grov       Audit log viewer
  |     +-- admin/user-lookup.grov     Wallet search/lookup
  |
  +-- templates/user/base.grov         User layout (extends base.grov)
        |-- Adds nav component
        |
        +-- user/signin.grov           Passphrase login form
        +-- user/dashboard.grov        User dashboard

templates/components/
  +-- alert.grov                       Error/success alert (props: error, success)
  +-- nav.grov                         Top navigation bar with sign-out
  +-- pagination.grov                  Pagination controls
```

---

## 11. Frontend Architecture

The frontend is server-rendered HTML with client-side JavaScript for auth flows and API calls. No build tools or frameworks.

### Static Files

| File | Purpose |
|------|---------|
| `static/css/main.css` | All application styles |
| `static/js/app.js` | User-side logic: signin, dashboard, token management |
| `static/js/admin.js` | Admin panel: login, CRUD operations, token management |
| `static/js/utils.js` | Shared utilities (API helpers, formatting, etc.) |

### Auth Token Flow (Client-Side)

**User flow:**
1. User enters passphrase on `/signin` page
2. JS sends `POST /api/v1/auth/signin` with passphrase
3. Server returns access token
4. JS stores token in `localStorage`
5. Subsequent API calls include `Authorization: Bearer <token>` header

**Admin flow:**
1. Admin enters password on `/admin/login` page
2. JS sends `POST /api/v1/admin/login` with password
3. Server generates admin session token (stored in memory), returns it
4. JS stores admin token in `localStorage`
5. Admin API calls include `Authorization: Bearer <admin-token>` header

### Important JS Convention

When parsing JSON responses, name the variable `result` not `data`:
```javascript
const result = await response.json();
// Correct: result.data.field
// Wrong: data.data.field (because WriteJSONResponse wraps in {data: ...})
```

---

## 12. Middleware Stack

Applied globally to all routes (in order):

| Middleware | Package | Purpose |
|------------|---------|---------|
| `middleware.Logger` | chi | Request logging |
| `middleware.Recoverer` | chi | Panic recovery |
| `middleware.RequestID` | chi | Injects `X-Request-Id` |
| `RateLimitMiddleware(120, time.Minute)` | custom | Token bucket: 120 requests/minute per client |

Route-group-specific middleware:

| Middleware | Applied To | Context Set |
|------------|-----------|-------------|
| `APIKeyMiddleware(pool)` | `/api/v1/providers/reference-codes`, `/api/v1/auth/signup` | `context.ProviderUUID` |
| `AccessTokenMiddleware(pool)` | All user API endpoints | `context.WalletUUID` |
| `AdminAuthMiddleware` | All `/api/v1/admin/` endpoints (except login) | None (just validates) |

---

## 13. Migration History

All migrations in `migrations/` directory, managed by Goose:

| Migration | Description |
|-----------|-------------|
| `20260312160318_inital_migration.sql` | Initial schema: wallet, currencies, balances, transactions tables |
| `20260312191500_add_balance_trigger.sql` | PostgreSQL trigger to auto-update balance on transaction insert |
| `20260324100000_add_providers_and_references.sql` | Providers and reference_codes tables |
| `20260325000000_update_wallet_schema.sql` | Wallet schema updates |
| `20260325120000_wallet_access_token_hash_not_null.sql` | Make access_token_hash NOT NULL |
| `20260326210000_add_access_tokens.sql` | Separate access_tokens table |
| `20260326220000_add_admin_audit_logs.sql` | Admin audit log table (JSONB details) |
| `20260327000000_add_wallet_token_hash_index.sql` | Index on wallet access_token_hash |
| `20260327000001_add_wallet_passphrase_hash.sql` | Add passphrase_hash column to wallet |
| `20260327000002_add_deleted_at_columns.sql` | Soft delete columns on wallet, currencies, balances |
| `20260405000000_add_transfer_id_to_transactions.sql` | Add transfer_id UUID column to transactions |
| `20260408000000_fix_balance_trigger.sql` | Fix balance trigger logic |

**Running migrations:**
```bash
./scripts/migrate.sh up|down|status
```

---

## 14. Seed System

**Source:** `internal/services/seed.go`

When `SEED_MODE=true`, the server seeds deterministic test data on startup. Idempotent -- checks if data already exists before creating.

### Seeded Data

| Entity | ID | Credentials |
|--------|-----|-------------|
| Wallet A | `00000000-0000-0000-0000-000000000001` | Passphrase: `test-user-alpha`, Token: `seed-access-token-wallet-alpha` |
| Wallet B | `00000000-0000-0000-0000-000000000002` | Passphrase: `test-user-beta`, Token: `seed-access-token-wallet-beta` |
| Provider | `00000000-0000-0000-0000-000000000003` | API Key: `seed-provider-api-key`, Name: `Seed Test Provider` |

### Seeded Balances

| Wallet | Currency | Amount |
|--------|----------|--------|
| Wallet A | USD | 10,000 |
| Wallet A | EUR | 5,000 |
| Wallet B | USD | 5,000 |
| Wallet B | EUR | 2,500 |

Balances are created via `deposit` transactions with reference `"seed"`. Access tokens have 365-day expiry. All credentials are logged to console on seed for testing reference.

---

## 15. Error Handling Conventions

### Handler Helpers

Defined in `internal/handlers/helpers.go`:

| Function | Purpose |
|----------|---------|
| `parseUUID(r, paramName)` | Parse UUID from URL param, returns `(uuid.UUID, error)` |
| `parseUUIDOrError(w, r, paramName, code, msg)` | Parse UUID and write 400 error if invalid |
| `handleNotFoundError(w, resourceType)` | Write 404 with resource type in message |
| `handleServiceError(w, err)` | Log error and write 500 |

### Error Wrapping

All error propagation uses `fmt.Errorf("context: %w", err)` for stack tracing. Service errors are wrapped with the operation context (e.g., `"get destination wallet: %w"`).

### HTTP Error Codes Used

| Code | When |
|------|------|
| 400 | Invalid request body, bad UUID format, validation failure |
| 401 | Missing or invalid auth credentials |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate currency code) |
| 422 | Business rule violation (insufficient balance, same wallet transfer) |
| 500 | Unexpected server error |

---

## 16. Testing

### Setup

- All tests in `tests/` package (separate from source code)
- `TestMain()` handles database setup and teardown
- Test database: `postgresql://postgres:postgres@localhost:5443/local_pg`
- `setupTestServer()` creates `httptest.Server` instances

### Helpers

- `createTestWallet()` / `deleteTestWallet()` -- wallet lifecycle
- `createTestCurrency()` / `deleteTestCurrency()` -- currency lifecycle
- Similar helpers for other entities
- Each test cleans up after itself

### Running Tests

```bash
go test ./...                        # All tests
go test -v ./tests -run TestName     # Single test
go test -race ./...                  # Race detection
go test -cover ./...                 # Coverage report
```

---

## 17. Build & Deployment

### Scripts

| Script | Purpose |
|--------|---------|
| `scripts/build-server.sh` | Compile Go binary to `tmp/kero-server` |
| `scripts/dev.sh [port]` | Run migrations + build + start server (default port 8090) |
| `scripts/migrate.sh up\|down\|status` | Run Goose migrations (requires goose CLI) |
| `scripts/deploy-postgres.sh` | Interactive Docker PostgreSQL setup |

### Server Configuration

| Setting | Value |
|---------|-------|
| Read timeout | 15 seconds |
| Write timeout | 15 seconds |
| Idle timeout | 60 seconds |
| Shutdown grace | 30 seconds |
| Rate limit | 120 requests/minute |

### Ignored Directories

- `tmp/` -- build output
- `_local_deploy/` -- Docker PostgreSQL data

---

## 18. Dependencies

### Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `go-chi/chi/v5` | v5.2.5 | HTTP router |
| `google/uuid` | v1.6.0 | UUID generation |
| `jackc/pgx/v5` | v5.8.0 | PostgreSQL driver + connection pool |
| `joho/godotenv` | v1.5.1 | .env file loading |
| `shopspring/decimal` | v1.4.0 | Arbitrary precision decimals |
| `wispberry-tech/go-common` | v0.0.4 | Structured logging, JSON response helpers |
| `wispberry-tech/grove` | v0.0.3 | Jinja2-like template engine |
| `golang.org/x/crypto` | v0.46.0 | bcrypt password hashing |

### Go Version

- Go 1.25.7

---

*End of specification.*
