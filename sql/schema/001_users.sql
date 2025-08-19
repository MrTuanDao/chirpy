-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    created_at TIMESTAMP, 
    updated_at TIMESTAMP, 
    email TEXT 
);

-- +goose Down
DROP TABLE users;