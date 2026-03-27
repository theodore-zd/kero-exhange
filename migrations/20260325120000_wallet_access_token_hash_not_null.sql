-- +goose Up
-- +goose StatementBegin

UPDATE wallet
SET access_token_hash = ''
WHERE access_token_hash IS NULL;

ALTER TABLE wallet
	ALTER COLUMN access_token_hash SET DEFAULT '',
	ALTER COLUMN access_token_hash SET NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE wallet
	ALTER COLUMN access_token_hash DROP NOT NULL,
	ALTER COLUMN access_token_hash DROP DEFAULT;

-- +goose StatementEnd
