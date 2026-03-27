# AGENTS.md - Coding Guidelines for kero-exchange

## Build Commands

### Build Binaries
```bash
# Build server binary (outputs to tmp/kero-server)
./scripts/build-server.sh

# Dev mode: migrate, build, and run server (default port 8090)
./scripts/dev.sh [port]
```

### Database Migrations
```bash
# Run migrations (requires goose: go install github.com/pressly/goose/v3/cmd/goose@latest)
./scripts/migrate.sh up
./scripts/migrate.sh down
./scripts/migrate.sh status
```

### Test Commands
```bash
# Run all tests (tests are in tests/ package)
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test file
go test -v ./tests -run TestWalletList

# Run a single test function
go test -v ./tests -run TestWalletGet_Success

# Run tests with race detection
go test -race ./...
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Tidy dependencies
go mod tidy
```

## Code Style Guidelines

### Import Organization
Imports are grouped in three sections with blank lines between:
1. Standard library (e.g., `context`, `fmt`, `net/http`)
2. Third-party packages (e.g., `github.com/...`)
3. Internal packages (e.g., `github.com/wispberry-tech/kero-exchange/...`)

Groups should be alphabetically sorted. Use `gofmt` to auto-format.

### Naming Conventions
- **Structs/Interfaces**: PascalCase (e.g., `WalletService`, `WalletHandler`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase (e.g., `walletSvc`, `pool`)
- **Constants**: PascalCase or UPPER_SNAKE_CASE for exported
- **Constructors**: `New[TypeName]` pattern (e.g., `NewWalletService(pool)`)
- **Receiver names**: Single lowercase letter matching type (e.g., `s` for `*WalletService`)

### Package Organization
```
cmd/server/     - HTTP server entry point
internal/config/ - Configuration loading
internal/db/    - Database access layer and models
internal/services/ - Business logic layer
internal/handlers/ - HTTP request handlers and web page handlers
templates/      - HTML templates for web interface
static/         - Static assets (CSS, JS)
migrations/     - Database migration files (SQL)
scripts/        - Build and deployment scripts
```

### Error Handling
- Check errors immediately after operations
- Use `%w` with `fmt.Errorf` for error wrapping
- Use `common.LogError` for logging with structured fields: `common.LogError("message", "key", value)`
- Use `common.WriteJSONError` for HTTP error responses with code and message
- Return nil checks with appropriate HTTP status codes (404, 400, etc.)

### Context Usage
- Pass `context.Context` as first parameter to service methods
- Use `r.Context()` from HTTP request
- Always propagate context through database calls

### Database Layer
- Use `pgx` library for PostgreSQL access
- Service layer calls db layer, which returns models
- Use `uuid.UUID` for primary keys (import `github.com/google/uuid`)
- Pagination: use `db.PaginationParams` with `Normalize()` and `db.Paginate()`
- Default page size: 20, max: 100

### HTTP Handler Patterns
- Handlers should be thin, delegating to services
- Use `common.WriteJSONResponse` for success responses
- Use separate Response DTOs to hide internal fields
- Format timestamps as RFC3339: `.UTC().Format("2006-01-02T15:04:05Z")`
- Register routes in `RegisterRoutes(r chi.Router)` method

### HTML Templates
- Use Go's `html/template` package for server-side rendering
- Base template (`templates/base.html`) defines common structure
- Page templates extend base using `{{define "content"}}` blocks
- Web handlers render templates using `renderTemplate(w, "name", data)`
- Keep JavaScript minimal - only for auth token management and API calls
- Use localStorage for storing access tokens

### Static Assets
- CSS files in `static/css/` directory
- JS files in `static/js/` directory
- Serve via `http.FileServer` with `/static/*` route
- Use plain CSS (no frameworks or preprocessors)
- Keep CSS simple and responsive
- Static files are auto-served by the HTTP server

### Configuration
- Load env files using `github.com/joho/godotenv`
- `postgres.env` for database connection
- `exchange.env` for application config
- Required env vars should return errors if missing
- Provide defaults for optional vars (PORT=8080, LOG_LEVEL=info)

### Logging
- Use `github.com/wispberry-tech/go-common` logging functions:
  - `common.LogInfo("message", "key", value)`
  - `common.LogError("message", "error", err)`
  - `common.SetLogLevel(level)` to set from config
- Initialize logger with `common.InitializeLogger()`

### JSON Responses
- Use `json:"field_name"` tags for consistent naming (snake_case)
- Omit empty fields with `omitempty` tag
- Use pointer types (`*string`, `*int64`) for optional fields
- Paginated responses: `{data: [...], meta: {page, page_size, total, total_pages}}`

### Testing Patterns
- Tests are in separate `tests/` package (not alongside code)
- Use `TestMain` for database setup and cleanup
- Create test helpers: `createTestWallet`, `createTestCurrency`, etc.
- Clean up test data in teardown: `deleteTestWallet`, etc.
- Use `setupTestServer` to create httptest servers
- Test database URL: `postgresql://postgres:postgres@localhost:5443/local_pg`

### Database Models
- Exported fields in internal/db package
- Include created_at, updated_at timestamps (time.Time)
- Use pointer types for nullable database columns (scan to `*string`, then assign to string if not nil)
- Use generic `PaginatedResult[T]` for paginated queries
- Use `PaginationParams` with `Normalize()` and `Offset()` for pagination
- Query functions return `nil, nil` for "not found" cases
