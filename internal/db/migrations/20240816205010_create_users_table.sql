-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE IF NOT EXISTS "users" (
    id SERIAL PRIMARY KEY,
    full_name varchar(25) NOT NULL,
    email CITEXT NOT NULL UNIQUE,
    password_hash bytea,
    is_verified bool DEFAULT false,
    auth_provider varchar(50) not null,
    auth_provider_id varchar(255),
    created_at timestamp with time zone NOT NULL DEFAULT (now())
);


-- +goose Down
DROP TABLE IF EXISTS "users";