-- name: CreateChirp :one
INSERT INTO chirps (created_at, updated_at, body, user_id)
VALUES (
	NOW(), 
	NOW(), 
	$1, 
	$2
)
RETURNING *;

-- name: GetAllChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetChirpByID :one
SELECT * from chirps where id = $1::uuid;