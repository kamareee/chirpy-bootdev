-- create a new database table called refresh_tokens with up/down migrations 
-- followings are the columns 
-- token: string, primary key
-- created_at: timestamp, not null 
-- updated_at: timestamp, not null
-- user_id: UUID, foreign key references users(id), not null
-- expires_at: timestamp, not null
-- revoked_at: timestamp, nullable

-- +goose Up
CREATE TABLE refresh_tokens (
    token VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL
);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;