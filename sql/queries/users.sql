-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: UpdateUser :one
update users
set email = $1, hashed_password = $2
where id = $3
RETURNING id, created_at, updated_at, email;

-- name: ResetUsers :exec
delete from users;

-- name: GetUserByEmail :one
select * from users where email = $1;