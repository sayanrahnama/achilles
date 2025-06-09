CREATE TABLE IF NOT EXISTS user_auth (
    id UUID PRIMARY KEY,
    hashed_password VARCHAR(255) NOT NULL
);