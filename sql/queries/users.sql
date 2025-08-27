-- name: CreateUser :one
INSERT INTO users (created_at, updated_at, email, hashed_password)
VALUES (
	NOW(), 
	NOW(), 
	$1, 
	$2
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email=$1;