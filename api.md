# Kero Exchange API Documentation

Base URL: `http://localhost:8090/api/v1`

## Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `X-API-Key` | Yes* | Provider API key for provider operations |
| `Authorization` | Yes** | Bearer token for authenticated user endpoints |

*Required for: signup, generate reference codes
**Required for: wallets, currencies, balances, transactions

## Common Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `page_size` | integer | 20 | Items per page (max 100) |

## Response Format

All responses follow this structure:

```json
{
  "data": { ... }
}
```

List responses include pagination metadata:

```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

Error responses:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

---

## Authentication

### Generate Reference Code

Generate a single one-time use reference code for wallet signup.
Always generates exactly 1 code valid for 1 hour. No parameters accepted.

**Endpoint:** `POST /api/v1/providers/reference-codes`

**Headers:**
- `X-API-Key` (required)

**Request Body:** None

**Response (201):**

```json
{
  "data": {
    "code": "A1B2C3D4E5F6G7H8",
    "expires_at": "2026-03-24T04:00:00Z"
  }
}
```

### Sign Up

Create a new wallet using a reference code.
Wallet UUID and secret passphrase are automatically generated.
The secret passphrase is returned **only once** during signup and cannot be recovered if lost.

**Endpoint:** `POST /api/v1/auth/signup`

**Headers:**
- `X-API-Key` (required)

**Request Body:**

```json
{
  "reference_code": "A1B2C3D4E5F6G7H8"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `reference_code` | string | Yes | Reference code from provider |

**Response (201):**

```json
{
  "data": {
    "access_token": "sha256:...",
    "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "secret_passphrase": "xK9mP2vQ4wR7sY3nAbCdEfGhIjKlMnOpQrStUvWxYz"
  }
}
```

**Important:** The `secret_passphrase` is returned **only once** during signup. If lost, the account cannot be recovered.

**Error Codes:**
- `MISSING_API_KEY` - X-API-Key header missing
- `INVALID_API_KEY` - Invalid API key
- `MISSING_REFERENCE_CODE` - Reference code not provided
- `INVALID_REFERENCE_CODE` - Reference code not found
- `REFERENCE_CODE_USED` - Reference code already used
- `REFERENCE_CODE_EXPIRED` - Reference code has expired

### Sign In

Authenticate an existing wallet using the secret passphrase.

**Endpoint:** `POST /api/v1/auth/signin`

**Request Body:**

```json
{
  "passphrase": "xK9mP2vQ4wR7sY3nAbCdEfGhIjKlMnOpQrStUvWxYz"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `passphrase` | string | Yes | Secret passphrase provided during signup |

**Response (200):**

```json
{
  "data": {
    "access_token": "sha256:...",
    "wallet_uuid": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Error Codes:**
- `MISSING_PASSPHRASE` - Passphrase not provided
- `WALLET_NOT_FOUND` - No wallet exists with this passphrase

---

## Wallets

### List Wallets

**Endpoint:** `GET /api/v1/wallets`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (optional) - Page number
- `page_size` (optional) - Items per page

**Response (200):**

```json
{
  "data": {
    "data": [
      {
        "uuid": "550e8400-e29b-41d4-a716-446655440000",
        "created_at": "2026-03-24T00:00:00Z",
        "updated_at": "2026-03-24T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

### Get Wallet

**Endpoint:** `GET /api/v1/wallets/{id}`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Path Parameters:**
- `id` (required) - Wallet UUID

**Response (200):**

```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2026-03-24T00:00:00Z",
    "updated_at": "2026-03-24T00:00:00Z"
  }
}
```

**Error Codes:**
- `MISSING_AUTH_TOKEN` - Authorization header missing
- `INVALID_ACCESS_TOKEN` - Invalid or expired access token
- `INVALID_UUID` - Invalid UUID format
- `NOT_FOUND` - Wallet not found

---

## Currencies

### List Currencies

**Endpoint:** `GET /api/v1/currencies`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (optional) - Page number
- `page_size` (optional) - Items per page

**Response (200):**

```json
{
  "data": {
    "data": [
      {
        "uuid": "550e8400-e29b-41d4-a716-446655440000",
        "code": "USD",
        "name": "US Dollar",
        "description": "United States Dollar",
        "created_at": "2026-03-24T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

### Get Currency

**Endpoint:** `GET /api/v1/currencies/{id}`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Path Parameters:**
- `id` (required) - Currency UUID

**Response (200):**

```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "code": "USD",
    "name": "US Dollar",
    "description": "United States Dollar",
    "created_at": "2026-03-24T00:00:00Z"
  }
}
```

### Get Currency by Code

**Endpoint:** `GET /api/v1/currencies/code/{code}`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Path Parameters:**
- `code` (required) - Currency code (e.g., "USD")

**Response (200):**

```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "code": "USD",
    "name": "US Dollar",
    "description": "United States Dollar",
    "created_at": "2026-03-24T00:00:00Z"
  }
}
```

**Error Codes:**
- `MISSING_AUTH_TOKEN` - Authorization header missing
- `INVALID_ACCESS_TOKEN` - Invalid or expired access token
- `INVALID_UUID` - Invalid UUID format
- `INVALID_CODE` - Currency code is required
- `NOT_FOUND` - Currency not found

---

## Balances

### List Balances

**Endpoint:** `GET /api/v1/balances`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (optional) - Page number
- `page_size` (optional) - Items per page
- `wallet_id` (optional) - Filter by wallet UUID
- `currency_id` (optional) - Filter by currency UUID

**Response (200):**

```json
{
  "data": {
    "data": [
      {
        "uuid": "550e8400-e29b-41d4-a716-446655440000",
        "wallet_id": "550e8400-e29b-41d4-a716-446655440001",
        "currency_id": "550e8400-e29b-41d4-a716-4466554400002",
        "balance": "1000.00",
        "updated_at": "2026-03-24T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

### Get Balance

**Endpoint:** `GET /api/v1/balances/{id}`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Path Parameters:**
- `id` (required) - Balance UUID

**Response (200):**

```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-446655440001",
    "currency_id": "550e8400-e29b-41d4-a716-4466554400002",
    "balance": "1000.00",
    "updated_at": "2026-03-24T00:00:00Z"
  }
}
```

**Error Codes:**
- `MISSING_AUTH_TOKEN` - Authorization header missing
- `INVALID_ACCESS_TOKEN` - Invalid or expired access token
- `INVALID_UUID` - Invalid UUID format
- `NOT_FOUND` - Balance not found

---

## Transactions

### List Transactions

**Endpoint:** `GET /api/v1/transactions`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (optional) - Page number
- `page_size` (optional) - Items per page
- `wallet_id` (optional) - Filter by wallet UUID
- `currency_id` (optional) - Filter by currency UUID
- `type` (optional) - Filter by transaction type (deposit, withdrawal, transfer)
- `start_date` (optional) - Filter by start date (RFC3339)
- `end_date` (optional) - Filter by end date (RFC3339)

**Response (200):**

```json
{
  "data": {
    "data": [
      {
        "uuid": "550e8400-e29b-41d4-a716-446655440000",
        "wallet_id": "550e8400-e29b-41d4-a716-4466554400001",
        "currency_id": "550e8400-e29b-41d4-a716-4466554400002",
        "amount": "100.00",
        "type": "deposit",
        "reference": "tx-123",
        "timestamp": "2026-03-24T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

### Get Transaction

**Endpoint:** `GET /api/v1/transactions/{id}`

**Headers:**
- `Authorization: Bearer <token>` (required)

**Path Parameters:**
- `id` (required) - Transaction UUID

**Response (200):**

```json
{
  "data": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "wallet_id": "550e8400-e29b-41d4-a716-4466554400001",
    "currency_id": "550e8400-e29b-41d4-a716-4466554400002",
    "amount": "100.00",
    "type": "deposit",
    "reference": "tx-123",
    "timestamp": "2026-03-24T00:00:00Z"
  }
}
```

**Error Codes:**
- `MISSING_AUTH_TOKEN` - Authorization header missing
- `INVALID_ACCESS_TOKEN` - Invalid or expired access token
- `INVALID_UUID` - Invalid UUID format
- `NOT_FOUND` - Transaction not found

---

## Health Check

### Health

**Endpoint:** `GET /health`

**Response (200):**

```json
{
  "data": {
    "status": "healthy"
  }
}
```

---

## Data Types

### UUID
Universally unique identifier. Format: `550e8400-e29b-41d4-a716-446655440000`

### Timestamp
ISO 8601 format with RFC3339. Example: `2026-03-24T00:00:00Z`

### Decimal
Amounts are returned as string representations to preserve precision. Example: `"1000.00"`

### Secret Passphrase
Secure random passphrase generated during signup. This is returned **only once** and cannot be recovered if lost. The passphrase is used for authentication and wallet recovery.
