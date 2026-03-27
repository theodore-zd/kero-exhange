# kero-exchange

Kero Exchange is a centralized currency exchange server with a web interface.

## Getting Started

### Prerequisites
- Go 1.25.7 or later
- PostgreSQL 14 or later

### Setup

1. Copy environment configuration files:
```bash
cp postgres.env.example postgres.env
cp exchange.env.example exchange.env
```

2. Configure your database connection in `postgres.env`

3. Configure application settings in `exchange.env`:
   - Set `ADMIN_PASSWORD` for admin UI access
   - Optionally configure default currency (`DEFAULT_CURRENCY_CODE`, `DEFAULT_CURRENCY_NAME`, `DEFAULT_CURRENCY_DESCRIPTION`)
   - Default currency is USD and will be auto-created on first server start

4. Run database migrations:
```bash
./scripts/migrate.sh up
```

5. Build and run the server:
```bash
./scripts/dev.sh
```

The server will start on `http://localhost:8090`

## Web Interface

Access the web interface at:
- Sign in: `http://localhost:8090/signin`
- Wallets: `http://localhost:8090/wallets`

The web interface uses Bearer token authentication (same as the API).

## API Documentation

See `api.md` for full API documentation.

## Development

### Build Commands
```bash
# Build server binary
./scripts/build-server.sh

# Run in development mode (migrate, build, run)
./scripts/dev.sh [port]
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# Run tests
go test ./...
```

### Project Structure
- `cmd/server/` - HTTP server entry point
- `internal/handlers/` - HTTP and web handlers
- `internal/services/` - Business logic
- `internal/db/` - Database models and queries
- `templates/` - HTML templates
- `static/` - CSS and JavaScript files
- `migrations/` - Database migrations

## License

See LICENSE file for details.
