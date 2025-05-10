CREATE TABLE IF NOT EXISTS user_auth (
    id UUID PRIMARY KEY,
    hashed_password VARCHAR(255) NOT NULL,
    CONSTRAINT fk_user_auth_user_id UNIQUE (id)
);