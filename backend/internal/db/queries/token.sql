-- name: CreateToken :one
INSERT INTO "tokens"  (
                      hash,
                      user_id,
                      expiry,
                      scope
) VALUES (
          $1,
          $2,
          $3,
          $4
) RETURNING *;


-- name: DeleteTokensForUser :exec
DELETE FROM "tokens"
WHERE scope = $1 AND user_id = $2;