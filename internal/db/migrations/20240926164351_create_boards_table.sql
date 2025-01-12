-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE IF NOT EXISTS "boards" (
    id SERIAL PRIMARY KEY,
    slug_id varchar(25) NOT NULL UNIQUE,
    name varchar(100) NOT NULL,
    owner_id BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    is_deleted bool NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS "board_pages" (
    id SERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL REFERENCES boards ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    is_deleted bool NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS "board_users" (
    board_id BIGINT NOT NULL REFERENCES boards ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    role varchar(20) NOT NULL DEFAULT 'viewer',
    PRIMARY KEY (board_id, user_id),
    CONSTRAINT chk_boards_users_role CHECK ( role IN ('editor', 'viewer') )
);


CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_boards_owner_id ON boards(owner_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_board_pages_board_id ON board_pages(board_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_board_users_user_id ON board_users(user_id);


-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_board_pages_board_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_boards_owner_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_board_users_user_id;
DROP TABLE IF EXISTS board_pages;
DROP TABLE IF EXISTS board_users;
DROP TABLE IF EXISTS boards;
