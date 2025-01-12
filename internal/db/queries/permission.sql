-- name: GetAllPermissionsForUser :many
SELECT permissions.code
FROM permissions
INNER JOIN user_permissions ON user_permissions.permission_id = permissions.id
INNER JOIN users ON user_permissions.user_id = users.id
WHERE users.id = $1;


-- name: CreatePermission :one
INSERT INTO "permissions" (code)
VALUES ($1)
RETURNING *;


-- name: AddPermissionForUser :one
INSERT INTO "user_permissions" (user_id, permission_id)
VALUES ($1, $2)
RETURNING *;


-- name: AddForUserWithCode :many
INSERT INTO "user_permissions" (user_id, permission_id)
SELECT @user_id, permissions.id FROM permissions WHERE permissions.code = ANY(@codes::text[])
RETURNING *;