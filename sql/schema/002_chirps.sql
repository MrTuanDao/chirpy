-- +goose Up
CREATE TABLE chirps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    created_at TIMESTAMP, 
    updated_at TIMESTAMP, 
    body TEXT, 
    user_id UUID references users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;