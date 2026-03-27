# Admin UI Documentation

## Setup

1. Set `ADMIN_PASSWORD` environment variable in `exchange.env`:
   ```
   ADMIN_PASSWORD=your-secure-password
   ```

2. Optionally configure default currency:
   ```
   DEFAULT_CURRENCY_CODE=USD
   DEFAULT_CURRENCY_NAME=US Dollar
   DEFAULT_CURRENCY_DESCRIPTION=Primary currency for the exchange
   ```

3. Ensure server is running:
   ```bash
   ./scripts/dev.sh
   ```

## Access

Navigate to `http://localhost:8090/admin/login` in your browser.

## Features

### Dashboard
- Overview of admin management options
- Navigation to providers, wallets, and currencies

### Providers Management
- **Create Provider**: Add new providers with name and API key
- **View Providers**: List all providers with UUID, name, and hashed API key
- **Edit API Key**: Rotate provider API keys (invalidates old key)
- **Delete Providers**: Remove providers (with confirmation)
- Pagination support

### Wallets Management
- **Create Wallet**: Generate new wallet with auto-generated passphrase and access token
- **View Wallets**: List all wallets with UUID, creation, and update timestamps
- **View Wallet Balances**: Expand wallet to see currency balances
- **Issue Currency**: Add currency funds to a wallet
- **Regenerate Passphrase**: Rotate wallet passphrase (invalidates old passphrase and access token)
- **Delete Wallets**: Remove wallets (with confirmation)
- Pagination support

### Currencies Management
- **Create Currency**: Add new fiat currencies with code, name, and optional description
- **View Currencies**: List all currencies with UUID, code, and name
- Pagination support

## Authentication

- Session-based authentication using localStorage
- Admin token stored in browser after successful login
- Logout button clears session

## Important Security Notes

### Passphrases
- Passphrases are shown **only once** after wallet creation or regeneration
- Save them securely when displayed
- They cannot be retrieved later
- Use "Regenerate Passphrase" if a passphrase is lost (this invalidates the old one)

### Provider API Keys
- API keys can be rotated at any time
- When rotating, the old key is immediately invalidated
- Shows hashed value in UI for identification

## Design

Minimal styling focusing on:
- Layout structure
- Visual hierarchy
- Modal dialogs for forms
- Alert messages for feedback
- Tables for data display
- Simple forms

## API Endpoints

### Authentication
- `POST /api/v1/admin/login` - Admin login (returns session token)

### Providers
- `POST /api/v1/admin/providers` - Create a new provider
  - Body: `{"name": "Provider Name", "api_key": "secret-key"}`
  - Response: Provider object with UUID, name, and API key hash
- `PUT /api/v1/admin/providers/{id}` - Update provider API key
  - Body: `{"api_key": "new-secret-key"}`
  - Response: 204 No Content
- `GET /api/v1/admin/providers` - List all providers (paginated)
- `DELETE /api/v1/admin/providers/{id}` - Delete a provider

### Wallets
- `POST /api/v1/admin/wallets` - Create a new wallet
  - Response: Wallet object with UUID, passphrase, and access token (shown only once)
- `POST /api/v1/admin/wallets/{id}/regenerate` - Regenerate wallet passphrase
  - Response: Wallet object with new passphrase and access token (shown only once)
- `POST /api/v1/admin/wallets/{id}/issue-currency` - Issue currency to a wallet
  - Body: `{"currency_id": "uuid", "amount": "100.00", "reference": "optional"}`
  - Response: Balance and transaction objects
- `GET /api/v1/admin/wallets` - List all wallets (paginated)
- `DELETE /api/v1/admin/wallets/{id}` - Delete a wallet

### Currencies
- `POST /api/v1/admin/currencies` - Create a new currency
  - Body: `{"code": "USD", "name": "US Dollar", "description": "Optional"}`
  - Response: Currency object with UUID, code, and name
- `GET /api/v1/admin/currencies` - List all currencies (paginated)

All admin API endpoints require admin token in Authorization header:
```
Authorization: Bearer <admin-token>
```

## User Flow Examples

### Creating a Provider
1. Navigate to `/admin/providers`
2. Click "Create Provider"
3. Enter provider name and API key
4. Click "Create"
5. Provider appears in list with hashed API key

### Rotating a Provider API Key
1. Navigate to `/admin/providers`
2. Click "Edit API Key" for provider
3. Enter new API key
4. Click "Update"
5. Warning: Old key is immediately invalidated

### Creating a Currency
1. Navigate to `/admin/currencies`
2. Click "Create Currency"
3. Enter currency code (e.g., USD), name (e.g., US Dollar), and optional description
4. Click "Create Currency"
5. Currency appears in list

### Creating a Wallet
1. Navigate to `/admin/wallets`
2. Click "Create Wallet"
3. Modal appears with:
   - Wallet UUID
   - **Passphrase (copy this now!)**
   - **Access Token (copy this now!)**
4. Close modal
5. Wallet appears in list

### Issuing Currency to a Wallet
1. Navigate to `/admin/wallets`
2. Click "Issue Currency" for the desired wallet
3. Select currency from dropdown
4. Enter amount
5. Optionally add a reference/note
6. Click "Issue Currency"
7. Currency is added to wallet balance

### Viewing Wallet Balances
1. Navigate to `/admin/wallets`
2. Click "Show currencies" for a wallet
3. Balances expand showing all currencies and amounts

### Regenerating a Wallet Passphrase
1. Navigate to `/admin/wallets`
2. Click "Regenerate Passphrase" for wallet
3. Confirm warning (old passphrase will be invalidated)
4. Modal appears with:
   - Wallet UUID
   - **New Passphrase (copy this now!)**
   - **New Access Token (copy this now!)**
5. Old passphrase is now invalid
