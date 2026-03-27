-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS providers (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reference_codes (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(16) NOT NULL UNIQUE,
    provider_id UUID NOT NULL REFERENCES providers(uuid) ON DELETE CASCADE,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reference_codes_code ON reference_codes(code);
CREATE INDEX idx_reference_codes_provider ON reference_codes(provider_id);
CREATE INDEX idx_reference_codes_expires ON reference_codes(expires_at) WHERE used_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS reference_codes;
DROP TABLE IF EXISTS providers;

-- +goose StatementEnd