#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BASE_URL="${1:-http://localhost:8090}"

# Load admin password from exchange.env if not set
if [ -z "${ADMIN_PASSWORD:-}" ]; then
    if [ -f "$ROOT_DIR/exchange.env" ]; then
        ADMIN_PASSWORD="$(grep -E '^ADMIN_PASSWORD=' "$ROOT_DIR/exchange.env" | cut -d= -f2-)"
    fi
fi

if [ -z "${ADMIN_PASSWORD:-}" ]; then
    echo "ERROR: ADMIN_PASSWORD not set and not found in exchange.env"
    exit 1
fi

PASS=0
FAIL=0
FAILURES=""

check_route() {
    local method="$1"
    local path="$2"
    local auth="${3:-}"
    local url="${BASE_URL}${path}"
    local label="${method} ${path}"

    local curl_args=(-s -o /dev/null -w "%{http_code}" -X "$method")
    if [ -n "$auth" ]; then
        curl_args+=(-H "Authorization: Bearer ${auth}")
    fi

    local status
    status=$(curl "${curl_args[@]}" "$url")

    if [ "$status" -ge 500 ]; then
        FAIL=$((FAIL + 1))
        FAILURES="${FAILURES}\n  FAIL ${label} → ${status}"
        printf "  \033[31mFAIL\033[0m %s → %s\n" "$label" "$status"
    elif [ "$status" -ge 400 ]; then
        # 4xx is expected for auth-protected routes without valid tokens, but flag it
        PASS=$((PASS + 1))
        printf "  \033[33m%s\033[0m  %s\n" "$status" "$label"
    else
        PASS=$((PASS + 1))
        printf "  \033[32m%s\033[0m  %s\n" "$status" "$label"
    fi
}

echo "=== Kero Exchange Page & API Tester ==="
echo "Base URL: ${BASE_URL}"
echo ""

# Get admin token
echo "Logging in as admin..."
LOGIN_RESP=$(curl -s -X POST "${BASE_URL}/api/v1/admin/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"${ADMIN_PASSWORD}\"}")

ADMIN_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
    echo "ERROR: Failed to get admin token. Response: ${LOGIN_RESP}"
    exit 1
fi
echo "Got admin token: ${ADMIN_TOKEN:0:8}..."
echo ""

# --- Public web pages (no auth) ---
echo "--- Public Web Pages ---"
check_route GET /health
check_route GET /signin
check_route GET /dashboard
check_route GET /wallets
check_route GET /transactions
check_route GET /transfer
echo ""

# --- Admin web pages (no server-side auth, JS handles it) ---
echo "--- Admin Web Pages ---"
check_route GET /admin/login
check_route GET /admin/dashboard
check_route GET /admin/providers
check_route GET /admin/providers/new
check_route GET /admin/wallets
check_route GET /admin/wallets/new
check_route GET /admin/currencies
check_route GET /admin/currencies/new
check_route GET /admin/transactions
check_route GET /admin/audit-log
check_route GET /admin/lookup
echo ""

# --- Admin API (Bearer token) ---
echo "--- Admin API ---"
check_route GET /api/v1/admin/providers "$ADMIN_TOKEN"
check_route GET /api/v1/admin/wallets "$ADMIN_TOKEN"
check_route GET "/api/v1/admin/wallets/search?q=test" "$ADMIN_TOKEN"
check_route GET /api/v1/admin/currencies "$ADMIN_TOKEN"
check_route GET /api/v1/admin/transactions "$ADMIN_TOKEN"
check_route GET /api/v1/admin/audit-logs "$ADMIN_TOKEN"
echo ""

# --- Summary ---
TOTAL=$((PASS + FAIL))
echo "=== Results: ${PASS}/${TOTAL} passed ==="
if [ "$FAIL" -gt 0 ]; then
    echo -e "\nFailures:${FAILURES}"
    exit 1
else
    echo "All routes OK!"
fi
