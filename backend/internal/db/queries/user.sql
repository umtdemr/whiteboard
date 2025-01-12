-- name: CreateUser :one
INSERT INTO "users"(
    email,
    password_hash,
    full_name,
    auth_provider,
    is_verified     
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) RETURNING *;


-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;


-- name: UpdateUser :one
UPDATE "users"
SET
    full_name = COALESCE(sqlc.narg(full_name), full_name),
    email = COALESCE(sqlc.narg(email), email),
    password_hash = COALESCE(sqlc.narg(password_hash), password_hash),
    is_verified = COALESCE(sqlc.narg(is_verified), is_verified),
    version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;


-- name: GetForToken :one
SELECT
    *
FROM users
INNER JOIN tokens
ON users.id = tokens.user_id
WHERE tokens.hash = $1
AND tokens.scope = $2
AND tokens.expiry > $3;