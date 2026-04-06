# Kero Exchange API Documentation

Base URL: `http://localhost:8080`

All API endpoints are prefixed with `/api/v1` unless otherwise noted.

---

## Table of Contents

- [Authentication](#authentication)
- [Credential Formats](#credential-formats)
- [Error Responses](#error-responses)
- [Pagination](#pagination)
- [Rate Limiting](#rate-limiting)
- [Public Endpoints](#public-endpoints)
- [Provider Endpoints (API Key)](#provider-endpoints-api-key)
- [User Endpoints (Access Token)](#user-endpoints-access-token)
- [Admin Endpoints (Admin Token)](#admin-endpoints-admin-token)
- [Provider Integration Guide](#provider-integration-guide)
- [Environment Variables](#environment-variables)

---

## Authentication

Kero Exchange uses three authentication mechanisms depending on the endpoint:

### 1. API Key Authentication

Used by **providers** (external services) for reference code generation and user signup.

```
X-API-Key: <provider-api-key>
```

API keys are issued by admins when creating a provider. The key is shown once at creation time and cannot be retrieved again.

### 2. Access Token Authentication

Used by **users** for all wallet, balance, transaction, and transfer operations.

```
Authorization: Bearer <access-token>
```

Access tokens are returned at signup and signin. Tokens expire after **24 hours**. The server tracks `last_used_at` for each token on every authenticated request.

### 3. Admin Token Authentication

Used by **admins** for all management operations.

```
Authorization: Bearer <admin-token>
```

Admin tokens are obtained by logging in with the admin password. Tokens expire after **24 hours**.

> **Note:** Admin tokens are stored **in-memory** on the server. They do not persist across server restarts — all admins must re-authenticate after a restart.

---

## Credential Formats

| Credential | Format | Example |
|------------|--------|---------|
| API Key | `kero_` prefix + 64 hex characters (70 chars total) | `kero_a1b2c3d4e5f6...` |
| Access Token | 64 hex characters | `a1b2c3d4e5f67890...` |
| Admin Token | 64 hex characters | `f0e1d2c3b4a59687...` |
| Secret Passphrase | Base64 URL-encoded 32 random bytes (~43 chars) | `dGhpcyBpcyBhIHRlc3Qgc3RyaW5n...` |
| Reference Code | 16 uppercase alphanumeric characters (`A-Z`, `0-9`) | `A1B2C3D4E5F6G7H8` |

> **Note:** Admin tokens and access tokens are **not JWTs**. They are opaque hex strings.

---

## Error Responses

All errors follow a consistent JSON format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

### Common Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `INVALID_REQUEST` | Malformed request body |
| 400 | `INVALID_UUID` | Invalid UUID path parameter |
| 401 | `MISSING_API_KEY` | Missing `X-API-Key` header on API key-protected routes |
| 401 | `INVALID_API_KEY` | `X-API-Key` provided but does not match any provider |
| 401 | `MISSING_AUTH_TOKEN` | Missing `Authorization` header on access token-protected routes |
| 401 | `INVALID_AUTH_FORMAT` | `Authorization` header is not in `Bearer <token>` format |
| 401 | `INVALID_ACCESS_TOKEN` | Access token does not match any wallet |
| 401 | `EXPIRED_ACCESS_TOKEN` | Access token has expired (>24 hours old) |
| 401 | `MISSING_ADMIN_TOKEN` | Missing `Authorization` header on admin-protected routes |
| 401 | `INVALID_ADMIN_TOKEN` | Admin token is invalid or expired |
| 404 | `NOT_FOUND` | Requested resource does not exist |
| 429 | `RATE_LIMITED` | Rate limit exceeded (120 requests/minute) |
| 500 | `INTERNAL_ERROR` | Server-side error |

---

## Pagination

All list endpoints support pagination via query parameters:

| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| `page` | 1 | — | Page number (1-indexed) |
| `page_size` | 20 | 100 | Items per page |

Invalid values are auto-corrected: `page` below 1 defaults to 1, `page_size` below 1 defaults to 20, and `page_size` above 100 is capped to 100.

Paginated responses include a `meta` object:

```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 57,
    "total_pages": 3
  }
}
```

---

## Rate Limiting

All endpoints are rate-limited to **120 requests per minute** per client IP.

The client IP is determined in this order:
1. `X-Forwarded-For` header (first value)
2. `X-Real-IP` header
3. `RemoteAddr` (direct connection IP)

When the limit is exceeded, the server returns:

```
HTTP 429 Too Many Requests
```

```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded"
  }
}
```

---

## Public Endpoints

### Health Check

```
GET /health
```

**Response** `200 OK`

```json
{
  "status": "healthy"
}
```

---

### Sign In

Authenticate an existing wallet using its secret passphrase.

```
POST /api/v1/auth/signin
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `passphrase` | string | Yes | The wallet's secret passphrase |

```json
{
  "passphrase": "dGhpcyBpcyBhIHRlc3Qgc3RyaW5nIGZvcg..."
}
```

**Response** `200 OK`

```json
{
  "access_token": "a1b2c3d4e5f67890abcdef1234567890a1b2c3d4e5f67890abcdef1234567890",
  "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_PASSPHRASE` | Passphrase field is empty |
| `WALLET_NOT_FOUND` | No wallet matches the passphrase |

---

### Admin Login

```
POST /api/v1/admin/login
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `password` | string | Yes | Admin password (set via `ADMIN_PASSWORD` env var) |

```json
{
  "password": "my-admin-password"
}
```

**Response** `200 OK`

```json
{
  "token": "f0e1d2c3b4a59687a1b2c3d4e5f67890f0e1d2c3b4a59687a1b2c3d4e5f67890"
}
```

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_PASSWORD` | Password field is empty |
| `INVALID_PASSWORD` | Wrong admin password |

---

## Provider Endpoints (API Key)

These endpoints require the `X-API-Key` header.

### Generate Reference Code

Create a single-use reference code that a user can redeem to create a wallet.

```
POST /api/v1/providers/reference-codes
```

**Headers**

```
X-API-Key: <provider-api-key>
```

**Request Body**: None

**Response** `201 Created`

```json
{
  "code": "A1B2C3D4E5F6G7H8",
  "expires_at": "2026-04-06T14:30:00Z"
}
```

Reference codes are 16 alphanumeric characters (uppercase + digits) and expire after **1 hour**.

---

### Sign Up (Create Wallet)

Redeem a reference code to create a new wallet.

```
POST /api/v1/auth/signup
```

**Headers**

```
X-API-Key: <provider-api-key>
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `reference_code` | string | Yes | A valid, unused reference code |

```json
{
  "reference_code": "A1B2C3D4E5F6G7H8"
}
```

**Response** `201 Created`

```json
{
  "access_token": "a1b2c3d4e5f67890abcdef1234567890a1b2c3d4e5f67890abcdef1234567890",
  "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "secret_passphrase": "dGhpcyBpcyBhIHRlc3Qgc3RyaW5nIGZvciBkb2Nz..."
}
```

> **Important:** The `secret_passphrase` is the **only way** for the user to sign back in. It is a base64url-encoded random string shown once and cannot be retrieved again. The user must store it securely.

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_REFERENCE_CODE` | Reference code field is empty |
| `INVALID_REFERENCE_CODE` | Reference code does not exist |
| `REFERENCE_CODE_USED` | Reference code has already been redeemed |
| `REFERENCE_CODE_EXPIRED` | Reference code has expired (>1 hour old) |

---

## User Endpoints (Access Token)

These endpoints require the `Authorization: Bearer <access-token>` header.

### Dashboard Summary

Get balance summary for the authenticated user's wallet.

```
GET /api/v1/dashboard
```

**Response** `200 OK`

> **Note:** `wallet_count` returns the **total system-wide wallet count**, not per-user.

```json
{
  "wallet_count": 42,
  "balances": [
    {
      "currency_code": "USD",
      "currency_name": "US Dollar",
      "total_amount": "1500.50000000"
    },
    {
      "currency_code": "EUR",
      "currency_name": "Euro",
      "total_amount": "200.00000000"
    }
  ]
}
```

---

### List Wallets

```
GET /api/v1/wallets?page=1&page_size=20
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2026-04-01T10:00:00Z",
      "updated_at": "2026-04-05T15:30:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

### Get Wallet

```
GET /api/v1/wallets/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Wallet UUID |

**Response** `200 OK`

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-04-01T10:00:00Z",
  "updated_at": "2026-04-05T15:30:00Z"
}
```

---

### List Currencies

```
GET /api/v1/currencies?page=1&page_size=20
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "660e8400-e29b-41d4-a716-446655440000",
      "code": "USD",
      "name": "US Dollar",
      "description": "United States Dollar",
      "created_at": "2026-04-01T10:00:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

### Get Currency by ID

```
GET /api/v1/currencies/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Currency UUID |

**Response** `200 OK`

```json
{
  "uuid": "660e8400-e29b-41d4-a716-446655440000",
  "code": "USD",
  "name": "US Dollar",
  "description": "United States Dollar",
  "created_at": "2026-04-01T10:00:00Z"
}
```

---

### Get Currency by Code

```
GET /api/v1/currencies/code/{code}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `code` | string | Currency code (e.g., `USD`, `EUR`) |

**Response** `200 OK` — same shape as Get Currency by ID.

---

### List Balances

```
GET /api/v1/balances?page=1&page_size=20
```

**Query Filters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `wallet_id` | UUID | Filter by wallet |
| `currency_id` | UUID | Filter by currency |

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "770e8400-e29b-41d4-a716-446655440000",
      "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
      "currency_id": "660e8400-e29b-41d4-a716-446655440000",
      "balance": "1500.50000000",
      "updated_at": "2026-04-05T15:30:00Z",
      "currency_code": "USD",
      "currency_name": "US Dollar"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

### Get Balance

```
GET /api/v1/balances/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Balance UUID |

**Response** `200 OK` — same shape as a single item in the balances list.

---

### List Transactions

```
GET /api/v1/transactions?page=1&page_size=20
```

**Query Filters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `wallet_id` | UUID | Filter by wallet |
| `currency_id` | UUID | Filter by currency |
| `type` | string | Filter by type: `deposit`, `withdrawal`, `transfer`, `admin_issued` |
| `start_date` | datetime | Filter transactions after this date |
| `end_date` | datetime | Filter transactions before this date |

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "880e8400-e29b-41d4-a716-446655440000",
      "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
      "currency_id": "660e8400-e29b-41d4-a716-446655440000",
      "amount": "100.00000000",
      "type": "deposit",
      "reference": "Initial deposit",
      "timestamp": "2026-04-05T15:30:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

### Transaction Types

| Type | Description |
|------|-------------|
| `deposit` | Funds added to wallet |
| `withdrawal` | Funds removed from wallet |
| `transfer` | Funds moved between wallets |
| `admin_issued` | Funds issued by an admin |

---

### Get Transaction

```
GET /api/v1/transactions/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Transaction UUID |

**Response** `200 OK` — same shape as a single item in the transactions list.

---

### Create Transfer

Transfer funds from the authenticated wallet to another wallet.

```
POST /api/v1/transfers
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination_wallet_id` | string (UUID) | Yes | Recipient wallet UUID |
| `currency_id` | string (UUID) | Yes | Currency to transfer |
| `amount` | string (decimal) | Yes | Amount to transfer, must be > 0 (e.g., `"100.50"`) |

> **Note:** Transfers do not support a `reference` field. Only `admin_issued` transactions can have references.

```json
{
  "destination_wallet_id": "550e8400-e29b-41d4-a716-446655440001",
  "currency_id": "660e8400-e29b-41d4-a716-446655440000",
  "amount": "100.50"
}
```

**Response** `201 Created`

```json
{
  "transfer_id": "990e8400-e29b-41d4-a716-446655440000",
  "debit": {
    "uuid": "aaa08400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
    "currency_id": "660e8400-e29b-41d4-a716-446655440000",
    "amount": "-100.50000000",
    "type": "transfer",
    "timestamp": "2026-04-06T12:00:00Z"
  },
  "credit": {
    "uuid": "bbb08400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-446655440001",
    "currency_id": "660e8400-e29b-41d4-a716-446655440000",
    "amount": "100.50000000",
    "type": "transfer",
    "timestamp": "2026-04-06T12:00:00Z"
  }
}
```

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_DESTINATION` | `destination_wallet_id` not provided |
| `INVALID_DESTINATION` | Invalid destination UUID |
| `MISSING_CURRENCY` | `currency_id` not provided |
| `INVALID_CURRENCY` | Invalid currency UUID |
| `MISSING_AMOUNT` | `amount` not provided |
| `INVALID_AMOUNT` | Amount is not a valid decimal |
| `SAME_WALLET` | Source and destination are the same wallet |
| `INSUFFICIENT_BALANCE` | Sender does not have enough funds |
| `DESTINATION_NOT_FOUND` | Destination wallet does not exist |
| `CURRENCY_NOT_FOUND` | Currency does not exist |

---

## Admin Endpoints (Admin Token)

These endpoints require the `Authorization: Bearer <admin-token>` header.

### Providers

#### Create Provider

```
POST /api/v1/admin/providers
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Provider display name |

```json
{
  "name": "My Payment App"
}
```

**Response** `201 Created`

```json
{
  "uuid": "110e8400-e29b-41d4-a716-446655440000",
  "api_key": "kero_a1b2c3d4e5f67890abcdef1234567890a1b2c3d4e5f67890abcdef12345678",
  "api_key_hash": "$2a$10$...",
  "name": "My Payment App",
  "created_at": "2026-04-06T10:00:00Z"
}
```

> **Important:** The `api_key` is returned **only at creation time**. It uses the format `kero_` + 64 hex characters. Store it securely — it cannot be retrieved again.

---

#### List Providers

```
GET /api/v1/admin/providers?page=1&page_size=20
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "110e8400-e29b-41d4-a716-446655440000",
      "api_key_hash": "$2a$10$...",
      "name": "My Payment App",
      "created_at": "2026-04-06T10:00:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

#### Update Provider API Key

```
PUT /api/v1/admin/providers/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Provider UUID |

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_key` | string | Yes | New API key to set |

```json
{
  "api_key": "new-api-key-value"
}
```

**Response** `204 No Content`

---

#### Delete Provider

```
DELETE /api/v1/admin/providers/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Provider UUID |

**Response** `204 No Content`

---

### Wallets

#### Create Wallet (Admin)

Create a wallet directly without a reference code.

```
POST /api/v1/admin/wallets
```

**Request Body**: None

**Response** `201 Created`

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "passphrase": "generated-secret-passphrase-...",
  "access_token": "a1b2c3d4e5f6...",
  "created_at": "2026-04-06T10:00:00Z",
  "updated_at": "2026-04-06T10:00:00Z"
}
```

---

#### List Wallets

```
GET /api/v1/admin/wallets?page=1&page_size=20
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2026-04-01T10:00:00Z",
      "updated_at": "2026-04-05T15:30:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

#### Search Wallets

```
GET /api/v1/admin/wallets/search?q=550e8400
```

**Query Parameters**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | Yes | Search query (matches against wallet UUID) |
| `page` | int | No | Page number |
| `page_size` | int | No | Items per page |

**Response** `200 OK` — same shape as List Wallets.

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_QUERY` | Search query `q` is required |

---

#### Regenerate Wallet Passphrase

Generate a new passphrase and access token for an existing wallet. Invalidates the old credentials.

```
POST /api/v1/admin/wallets/{id}/regenerate
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Wallet UUID |

**Response** `200 OK`

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "passphrase": "new-secret-passphrase-...",
  "access_token": "new-access-token-...",
  "created_at": "2026-04-01T10:00:00Z",
  "updated_at": "2026-04-06T10:00:00Z"
}
```

---

#### Delete Wallet

Soft-deletes a wallet (sets `deleted_at`).

```
DELETE /api/v1/admin/wallets/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Wallet UUID |

**Response** `204 No Content`

---

#### Get Wallet Balances

```
GET /api/v1/admin/wallets/{id}/balances?page=1&page_size=20
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Wallet UUID |

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "770e8400-e29b-41d4-a716-446655440000",
      "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
      "currency_id": "660e8400-e29b-41d4-a716-446655440000",
      "balance": "1500.50000000",
      "updated_at": "2026-04-05T15:30:00Z",
      "currency_code": "USD",
      "currency_name": "US Dollar"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

#### Issue Currency to Wallet

Deposit funds into a wallet. Creates a balance if one doesn't exist for that currency.

```
POST /api/v1/admin/wallets/{id}/issue-currency
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Wallet UUID |

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `currency_id` | string (UUID) | Yes | Currency to issue |
| `amount` | string (decimal) | Yes | Amount to deposit (e.g., `"500.00"`) |
| `reference` | string | No | Optional reference note |

```json
{
  "currency_id": "660e8400-e29b-41d4-a716-446655440000",
  "amount": "500.00",
  "reference": "Welcome bonus"
}
```

**Response** `201 Created`

```json
{
  "balance": {
    "uuid": "770e8400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
    "currency_id": "660e8400-e29b-41d4-a716-446655440000",
    "balance": "500.00000000",
    "updated_at": "2026-04-06T12:00:00Z",
    "currency_code": "USD",
    "currency_name": "US Dollar"
  },
  "transaction": {
    "uuid": "880e8400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
    "currency_id": "660e8400-e29b-41d4-a716-446655440000",
    "amount": "500.00000000",
    "type": "admin_issued",
    "reference": "Welcome bonus",
    "timestamp": "2026-04-06T12:00:00Z"
  },
  "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_CURRENCY_ID` | `currency_id` not provided |
| `INVALID_CURRENCY_ID` | Invalid currency UUID format |
| `MISSING_AMOUNT` | `amount` not provided |

---

### Currencies

#### Create Currency

```
POST /api/v1/admin/currencies
```

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | Yes | Currency code (max 8 characters, e.g., `USD`) |
| `name` | string | Yes | Display name |
| `description` | string | No | Optional description |

```json
{
  "code": "USD",
  "name": "US Dollar",
  "description": "United States Dollar"
}
```

**Response** `201 Created`

```json
{
  "uuid": "660e8400-e29b-41d4-a716-446655440000",
  "code": "USD",
  "name": "US Dollar",
  "description": "United States Dollar",
  "created_at": "2026-04-06T10:00:00Z"
}
```

**Errors**

| Code | Description |
|------|-------------|
| `MISSING_CODE` | Currency code is required |
| `MISSING_NAME` | Currency name is required |
| `INVALID_CODE` | Currency code exceeds 8 characters |

---

#### List Currencies

```
GET /api/v1/admin/currencies?page=1&page_size=20
```

**Response** `200 OK` — same shape as user currency list.

---

#### Delete Currency

Soft-deletes a currency. When successful, **all balances and transactions** in that currency are also soft-deleted in cascade. Fails if any wallets hold a non-zero balance in this currency.

```
DELETE /api/v1/admin/currencies/{id}
```

**Path Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Currency UUID |

**Response** `204 No Content`

**Errors**

| Code | Description |
|------|-------------|
| `CURRENCY_IN_USE` | Cannot delete — wallets hold balances in this currency |

---

### Transactions

#### List Transactions (Admin)

```
GET /api/v1/admin/transactions?page=1&page_size=20
```

**Query Filters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `wallet_id` | UUID | Filter by wallet |
| `currency_id` | UUID | Filter by currency |
| `type` | string | Filter by type: `deposit`, `withdrawal`, `transfer`, `admin_issued` |
| `from` | datetime | Filter transactions after this date |
| `to` | datetime | Filter transactions before this date |

> **Note:** The admin transaction list uses `from`/`to` for date filters, while the user list uses `start_date`/`end_date`.

**Response** `200 OK` — same shape as user transaction list.

---

#### Get Transaction (Admin)

```
GET /api/v1/admin/transactions/{id}
```

**Response** `200 OK` — same shape as user Get Transaction.

---

### Audit Logs

#### List Audit Logs

All admin actions are recorded in the audit log.

```
GET /api/v1/admin/audit-logs?page=1&page_size=20
```

**Query Filters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `action` | string | Filter by action type |
| `entity_type` | string | Filter by entity type (e.g., `provider`, `wallet`, `currency`) |

**Response** `200 OK`

```json
{
  "data": [
    {
      "uuid": "cc0e8400-e29b-41d4-a716-446655440000",
      "action": "create_provider",
      "entity_type": "provider",
      "entity_id": "110e8400-e29b-41d4-a716-446655440000",
      "details": {
        "name": "My Payment App"
      },
      "admin_user": "admin",
      "ip_address": "192.168.1.1",
      "user_agent": "curl/7.68.0",
      "created_at": "2026-04-06T10:00:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

## Provider Integration Guide

This section walks through the complete flow of integrating with Kero Exchange as a provider.

### 1. Get Your API Key

An admin creates your provider account and gives you an API key:

```bash
# Admin creates a provider (admin does this)
curl -X POST http://localhost:8080/api/v1/admin/providers \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Payment App"}'

# Response includes the api_key — store it securely
```

### 2. Generate a Reference Code for a New User

When a user wants to create an account through your app:

```bash
curl -X POST http://localhost:8080/api/v1/providers/reference-codes \
  -H "X-API-Key: kero_a1b2c3d4e5f67890..."
```

```json
{
  "code": "A1B2C3D4E5F6G7H8",
  "expires_at": "2026-04-06T14:30:00Z"
}
```

The code expires in 1 hour. Generate it just before the user needs it.

### 3. Sign Up the User

Use the reference code to create the user's wallet:

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "X-API-Key: kero_a1b2c3d4e5f67890..." \
  -H "Content-Type: application/json" \
  -d '{"reference_code": "A1B2C3D4E5F6G7H8"}'
```

```json
{
  "access_token": "a1b2c3d4e5f67890abcdef1234567890a1b2c3d4e5f67890abcdef1234567890",
  "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "secret_passphrase": "dGhpcyBpcyBhIHRlc3Qgc3RyaW5nIGZvciBkb2Nz..."
}
```

**You must securely deliver the `secret_passphrase` to the user.** This is their only way to sign back in — it's a base64url-encoded random string. The access token is for immediate API use.

### 4. User Signs In Later

When the user returns and needs a fresh access token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"passphrase": "dGhpcyBpcyBhIHRlc3Qgc3RyaW5nIGZvciBkb2Nz..."}'
```

```json
{
  "access_token": "b2c3d4e5f67890abcdef1234567890a1b2c3d4e5f67890abcdef1234567890ab",
  "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 5. Use the Wallet

With the access token, the user (or your app on their behalf) can:

**Check balances:**

```bash
curl http://localhost:8080/api/v1/balances?wallet_id=550e8400-... \
  -H "Authorization: Bearer <access-token>"
```

**View transaction history:**

```bash
curl http://localhost:8080/api/v1/transactions?wallet_id=550e8400-... \
  -H "Authorization: Bearer <access-token>"
```

**Transfer funds to another wallet:**

```bash
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "destination_wallet_id": "another-wallet-uuid",
    "currency_id": "currency-uuid",
    "amount": "50.00"
  }'
```

### Complete Flow Diagram

```
Admin creates Provider  ──→  Provider gets API key
                                    │
                                    ▼
                         Provider generates reference code
                                    │
                                    ▼
                         User redeems code (signup)
                                    │
                                    ▼
                         User receives: wallet_uuid,
                         access_token, secret_passphrase
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
            User stores passphrase          User uses access_token
            (for future sign-in)            to call API endpoints
                                                    │
                                    ┌───────────────┼───────────────┐
                                    ▼               ▼               ▼
                              Check balances  View transactions  Transfer funds
```

### Financial Precision

All monetary amounts use `DECIMAL(20,8)` precision. Amounts in API responses are returned as strings (e.g., `"1500.50000000"`) to avoid floating-point issues. Always send amounts as strings in requests.

### Soft Deletes

Wallets, currencies, and balances use soft deletes (a `deleted_at` timestamp is set rather than removing the row). The `deleted_at` field is **not** exposed in API responses — soft-deleted resources simply stop appearing in list/get results.

---

## Environment Variables

The server is configured via two env files:

### postgres.env

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string (e.g., `postgresql://user:pass@host:5432/dbname`) |
| `GOOSE_DRIVER` | Yes | Set to `postgres` (used by migration tool) |
| `GOOSE_DBSTRING` | Yes | Same as `DATABASE_URL` (used by migration tool) |

### exchange.env

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ADMIN_PASSWORD` | Yes | — | Password for admin login |
| `PORT` | No | `8080` | HTTP server port |
| `LOG_LEVEL` | No | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |
| `DEFAULT_CURRENCY_CODE` | No | `USD` | Currency code auto-created on first boot |
| `DEFAULT_CURRENCY_NAME` | No | `US Dollar` | Name for the default currency |
| `DEFAULT_CURRENCY_DESCRIPTION` | No | — | Description for the default currency |
| `SEED_MODE` | No | `false` | Set to `true` to seed test data on startup |
