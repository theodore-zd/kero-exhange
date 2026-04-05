# Frontend Redesign: Grove Migration, User Flows & Dev Seed Mode

**Date:** 2026-04-05
**Scope:** Full rewrite of all frontend templates, expanded user/admin functionality, dev seed mode

## Overview

Replace all Go `html/template` templates with Grove (`github.com/wispberry-tech/grove`), a bytecode-compiled Jinja2-style template engine. Redesign all pages with a terminal/monospace aesthetic. Expand user flows to include a full dashboard with transfers. Expand admin with transaction browser, audit log, and wallet lookup. Add a dev seed mode for automatic test environment setup.

## 1. Template Engine & Rendering Layer

### Grove Integration

- Add `github.com/wispberry-tech/grove` as a Go module dependency
- Create a `grove.Engine` instance at startup, shared across all handlers
- Templates use `.grov` files with Grove's Jinja2-style syntax
- Auto-escaping stays on (Grove default)

### Layout System

- `base.grov` root layout using Grove's `extends`/`block` inheritance
- Blocks: `title`, `content`, `scripts`
- Grove's `{% asset %}` and `{% hoist %}` for per-page CSS/JS dependency management

### Handler Rendering Pattern

- Replace existing `renderTemplate()` methods with a shared helper that calls `engine.RenderTemplate(ctx, templateString, data)` and writes `RenderResult.Body` + collected assets to the response
- Handlers remain thin: parse request, call service, render template with data

### Template File Structure

```
templates/
  base.grov                # Root layout (html shell, head, body)
  components/              # Reusable Grove components (pagination, table, nav, alert)
  user/                    # User-facing pages
    signin.grov
    dashboard.grov
    wallets.grov
    wallet-detail.grov
    transactions.grov
    transfer.grov
  admin/                   # Admin pages
    login.grov
    dashboard.grov
    providers.grov
    provider-form.grov
    wallets.grov
    wallet-detail.grov
    wallet-form.grov
    currencies.grov
    currency-form.grov
    transactions.grov
    audit-log.grov
    user-lookup.grov
```

## 2. User Flows & Pages

### Sign In (`/signin`)

- Single page with passphrase input field
- On success: store access token + wallet UUID in localStorage, redirect to `/dashboard`

### User Dashboard (`/dashboard`)

- Landing page after sign in
- Summary: wallet count, total balances across currencies
- Quick links to wallets, transactions, transfer

### Wallets (`/wallets`)

- Paginated list of user's wallets
- Each wallet shows UUID, creation date
- Click wallet to see detail page

### Wallet Detail (`/wallets/{id}`)

- Wallet info and all balances
- Recent transactions for that wallet
- Link to initiate a transfer from this wallet

### Transactions (`/transactions`)

- Paginated list of all transactions across user's wallets
- Filterable by wallet, currency, type (deposit, withdrawal, transfer)
- Columns: date, type, amount, currency, from/to wallet, status

### Transfer (`/transfer`)

- Form: select source wallet, enter destination wallet UUID, select currency, enter amount
- Client-side confirmation step (JS confirm dialog showing summary before submit, not a separate page)
- On success: show transaction details inline
- Client-side balance validation (server-side as source of truth)

### User Navigation

- Top horizontal bar: Dashboard, Wallets, Transactions, Transfer, Sign Out
- No sidebar for user pages
- Current page highlighted in nav

## 3. Admin Flows & Pages

### Admin Login (`/admin/login`)

- Password input, token stored in localStorage
- Same auth mechanism as today (in-memory AdminTokenStore, 24h expiry)

### Admin Dashboard (`/admin/dashboard`)

- Overview stats: total wallets, providers, currencies, recent transaction count
- Quick links to all admin sections

### Providers (`/admin/providers`)

- Paginated list with name, status, creation date
- Create new provider (shows API key once after creation)
- Edit provider API key
- Delete provider

### Wallets (`/admin/wallets`)

- Paginated list with UUID, creation date, balance summary
- Create new wallet (shows passphrase + access token once after creation)
- Wallet detail: full info, all balances, all transactions
- Actions: regenerate passphrase, issue currency, delete wallet

### Currencies (`/admin/currencies`)

- Paginated list with code, name, description
- Create new currency
- Delete currency

### Transactions (`/admin/transactions`) — NEW

- Paginated browser of all transactions system-wide
- Filterable by wallet, currency, type, date range
- Read-only view

### Audit Log (`/admin/audit-log`) — NEW

- Paginated list of all admin actions
- Columns: timestamp, actor, action, target resource, details
- Filterable by action type, date range

### Wallet Lookup (`/admin/lookup`) — NEW

- Search by wallet UUID or partial match
- Results show wallet info, balances, recent transactions
- Quick links to wallet detail, issue currency

### Admin Navigation

- Sidebar: Dashboard, Providers, Wallets, Currencies, Transactions, Audit Log, Lookup, Logout
- Current section highlighted

## 4. New API Endpoints & Backend

### New Endpoints

**Transfer (access token protected):**
```
POST /api/v1/transfers
```
Body: `{source_wallet_id, destination_wallet_id, currency_id, amount}`

**Admin transactions (admin protected):**
```
GET  /api/v1/admin/transactions
GET  /api/v1/admin/transactions/{id}
```

**Admin audit log (admin protected):**
```
GET  /api/v1/admin/audit-logs
```

**Admin wallet lookup (admin protected):**
```
GET  /api/v1/admin/wallets/search?q=...
```

**User dashboard summary (access token protected):**
```
GET  /api/v1/dashboard
```
Returns: `{wallet_count, balances: [{currency_code, currency_name, total_amount}], recent_transactions: [...]}`

### Transfer Service Logic

1. Validate source wallet belongs to authenticated user
2. Validate destination wallet exists
3. Validate currency exists and source has sufficient balance
4. Debit source, credit destination in a single database transaction
5. Create two transaction records (debit + credit) linked by a shared `transfer_id` field (new UUID column on transactions table)
6. Return both transaction records with the transfer_id

### Filter Parameters

All list endpoints support:
- `wallet_id` — filter by wallet
- `currency_id` — filter by currency
- `type` — filter by transaction type
- `from` / `to` — date range (RFC3339)
- `page`, `page_size` — standard pagination

## 5. Dev Seed Mode

### Activation

- Environment variable `SEED_MODE=true` in `exchange.env`
- On startup, checks if seed data exists (by known deterministic UUID); creates only if missing
- Logs credentials on startup

### Seed Data

- **Currencies:** USD (US Dollar), EUR (Euro)
- **Wallet A:** passphrase `test-user-alpha`, 10,000 USD, 5,000 EUR
- **Wallet B:** passphrase `test-user-beta`, 5,000 USD, 2,500 EUR
- **Provider:** known API key for testing reference code / signup flows
- **UUIDs:** Deterministic (e.g. `00000000-0000-0000-0000-000000000001`) for predictable credentials

### Startup Output

```
[SEED] Dev seed mode enabled
[SEED] Test Wallet A: passphrase=test-user-alpha uuid=00000000-...0001
[SEED] Test Wallet B: passphrase=test-user-beta  uuid=00000000-...0002
[SEED] Test Provider: api-key=test-provider-key-xxx
```

### Safety

- Only runs when `SEED_MODE=true` — disabled by default
- Idempotent — safe across restarts
- Obviously fake values to avoid confusion with real data

## 6. Static Assets & CSS

### CSS

- Single `static/css/main.css`, no framework
- Monospace font stack: `"SF Mono", "Cascadia Code", "Fira Code", Consolas, monospace`
- Dark theme: background `#1a1a1a`, text `#e0e0e0`, accent green `#00ff88`, error red `#ff4444`, link cyan `#00cccc`
- No shadows, rounded corners, or gradients
- Plain tables with subtle row separators
- Simple bordered inputs, minimal padding
- Responsive: single column on narrow screens

### JavaScript

- `static/js/app.js` — user-side auth helpers (token storage, authenticated fetch, sign out)
- `static/js/admin.js` — admin-side auth helpers
- Transfer form validation inline or in small `transfer.js`
- No build tools, no frameworks, vanilla JS
- Page-specific scripts via Grove's `{% hoist %}`
