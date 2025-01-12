-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY idx_auth_provider ON users(auth_provider, auth_provider_id);

-- +goose Down
DROP INDEX IF EXISTS idx_auth_provider;
