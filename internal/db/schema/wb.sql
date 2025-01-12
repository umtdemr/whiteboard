CREATE TABLE IF NOT EXISTS "users" (
    id SERIAL PRIMARY KEY,
    full_name varchar(25) NOT NULL,
    email CITEXT NOT NULL UNIQUE,
    password_hash bytea,
    is_verified bool DEFAULT false,
    auth_provider varchar(50) not null,
    auth_provider_id varchar(255),
    created_at timestamp with time zone NOT NULL DEFAULT (now()),
    version integer NOT NULL DEFAULT 1
);


CREATE TABLE IF NOT EXISTS "tokens" (
    hash bytea PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL,
    scope text NOT NULL
);


CREATE TABLE IF NOT EXISTS "permissions" (
    id bigserial PRIMARY KEY,
    code text NOT NULL
);


CREATE TABLE IF NOT EXISTS "user_permissions" (
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    permission_id bigint NOT NULL REFERENCES permissions ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

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