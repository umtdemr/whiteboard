-- name: CreateBoard :one
INSERT INTO "boards" (
    slug_id,
    name,
    owner_id
) VALUES (
    $1,
    $2,
    $3
) RETURNING *;

-- name: CreateBoardPage :one
INSERT INTO "board_pages" (
    board_id,
    name
) VALUES (
     $1,
     $2
) RETURNING *;

-- name: AddToBoardUsers :one
INSERT INTO "board_users" (board_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;


-- name: GetBoardById :one
SELECT *
FROM boards
WHERE id = $1;

-- name: GetAllBoardsForUser :many
SELECT b.id, b.slug_id, b.owner_id, b.created_at, b.is_deleted, b.name,
       CASE
           WHEN b.owner_id = $1 THEN TRUE
           ELSE FALSE
           END AS is_owner
FROM boards b
 JOIN board_users bu ON bu.board_id = b.id AND bu.user_id = $1
WHERE b.is_deleted = FALSE
ORDER BY b.created_at DESC;

-- name: GetBoardBySlugId :one 
SELECT b.id, b.slug_id, b.owner_id, b.created_at, b.is_deleted, b.name,
       CASE
           WHEN b.owner_id = $1 THEN TRUE
           ELSE FALSE
           END AS is_owner
FROM boards b
JOIN board_users bu ON bu.board_id = b.id AND bu.user_id = $1
WHERE b.is_deleted = FALSE and b.slug_id = $2;


-- name: GetBoardPageByBoardId :many
SELECT name, id, created_at
FROM board_pages
WHERE board_id = $1 and is_deleted = false;


-- name: GetBoardUsers :many
SELECT u.id, u.full_name, u.email, bu.role FROM users u
LEFT JOIN board_users bu ON bu.user_id = u.id
WHERE bu.board_id = $1;