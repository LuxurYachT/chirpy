-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: GetChirps :many
select * from chirps;

-- name: GetChirp :one
select * from chirps where id = $1;

-- name: GetChirpOwner :one
select user_id from chirps where id = $1;

-- name: DeleteChirp :exec
delete from chirps where id = $1;